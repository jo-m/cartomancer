// Package jobs implements an async job queue.
// The [db] package is used for persistence between restarts.
// Periodically scheduled jobs are also supported.
// Jobs are run according to at-least-once semantics.
//
// The interface is partially inspired by https://riverqueue.com/docs.
package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/jo-m/goweb/internal/pkg/db"
	"github.com/jo-m/goweb/internal/pkg/logg"
)

// Args is the interface job args have to implement.
// In addition, the implementing struct must be JSON serializable,
// and all members must be publicly visible.
type Args interface {
	// Kind must return a constant, unique string identifying the job type.
	Kind() string
}

// Job is the interface to define custom jobs.
type Job[T Args] interface {
	// Run must be implemented to run the job once.
	// Returning nil means successful execution.
	// Returning an error means that the job will be retried (if maxAttemps is set to > 1).
	// Implementations are responsible maintain reasonable timeout for execution themselves.
	Run(ctx context.Context, args T) error
}

// JobsConfig is the configuration for [Workers].
// It has struct tags compatible with [github.com/alexflint/go-arg].
//
//revive:disable:exported Naming necessary for struct embedding.
type JobsConfig struct {
	// MaxParallel is the maximum number of jobs that can run in parallel.
	// It defaults to runtime.NumCPU() if zero.
	MaxParallel uint `arg:"--jobs-max-parallel,env:JOBS_MAX_PARALLEL" default:"0" help:"Maximum number of parallel jobs" placeholder:"N"`
	// AutoCleanupPeriod is the period at which old jobs will be cleared from the database.
	// Disabled if set to 0.
	AutoCleanupPeriod time.Duration `arg:"--jobs-auto-cleanup-period,env:JOBS_AUTO_CLEANUP_PERIOD" default:"0" help:"Period at which old jobs will be cleared from the database" placeholder:"DUR"`
	// AutoCleanupMinAge is the time to wait until jobs are cleared from the database.
	AutoCleanupMinAge time.Duration `arg:"--jobs-auto-cleanup-min-age,env:JOBS_AUTO_CLEANUP_MIN_AGE" default:"0" help:"Time to wait after a job has finished to clear it from the database" placeholder:"DUR"`
}

type decodeAndWorkFunc func(ctx context.Context, args json.RawMessage) error

// Workers runs jobs.
// Use [NewWorkers] create an instance.
type Workers struct {
	d       *db.DB
	c       JobsConfig
	running atomic.Bool
	w       map[string]decodeAndWorkFunc
}

func sqlTimeNow() sql.NullTime {
	return sql.NullTime{Time: time.Now(), Valid: true}
}

// NewWorkers creates a new workers instance.
// There can only be one instance per process and database.
// Use [jobs.RegisterJob] to register new jobs on the worker.
// Use [jobs.Submit] and [jobs.Periodic] on a submitter (see [*Workers.Submitter]) to submit jobs to be run.
func NewWorkers(ctx context.Context, d *db.DB, c JobsConfig) (*Workers, error) {
	if err := ensureSingleInstance(ctx, d); err != nil {
		return nil, err
	}

	n, err := d.QueryRW().SetJobsAborted(ctx, db.SetJobsAbortedParams{
		FinishedAt: sqlTimeNow(),
		OurPID:     sql.NullInt64{Valid: true, Int64: randomID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to mark aborted jobs: %w", err)
	} else if n > 0 {
		logg.Warn(ctx, "Aborted jobs from previous proc", "count", n)
	}

	w := &Workers{
		d:       d,
		c:       c,
		running: atomic.Bool{},
		w:       map[string]decodeAndWorkFunc{},
	}

	if c.AutoCleanupPeriod != 0 {
		MustRegisterJob(w, &cleaner{d: d})
		s := w.Submitter()
		Periodic(ctx, s, cleanerArgs{
			MinAge: c.AutoCleanupMinAge,
		}, c.AutoCleanupPeriod)
	}

	return w, nil
}

func checkArgsType[T Args]() error {
	typ := reflect.TypeFor[T]()
	typeName := fmt.Sprint(typ.PkgPath(), ".", typ.Name())

	var args T
	data, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("'%s' is not JSON serializable: %w", typeName, err)
	}

	var target T
	err = json.Unmarshal(data, &target)
	if err != nil {
		return fmt.Errorf("'%s' is not JSON deserializable: %w", typeName, err)
	}

	if reflect.TypeOf(args).Kind() != reflect.Struct {
		return fmt.Errorf("'%s' is not a struct type", typeName)
	}

	for i := range typ.NumField() {
		field := typ.Field(i)
		if !field.IsExported() {
			return fmt.Errorf("field '%s' on '%s' is not exported", field.Name, typeName)
		}

		if field.Type.Kind() == reflect.Interface {
			return fmt.Errorf("field '%s' on '%s' is an interface", field.Name, typeName)
		}
	}

	kind := args.Kind()
	if kind == "" {
		return fmt.Errorf("%s returns empty Kind()", typeName)
	}

	for range 100 {
		k := args.Kind()
		if k != kind {
			return fmt.Errorf("%s returns different Kind() values: %q != %q", typeName, kind, k)
		}
	}

	return nil
}

// RegisterJob registers a new job.
// After registration, jobs of this type can be submitted for running.
func RegisterJob[T Args](w *Workers, worker Job[T]) error {
	if w.running.Load() {
		return errors.New("already running")
	}

	err := checkArgsType[T]()
	if err != nil {
		return err
	}

	var args T
	kind := args.Kind()

	if _, ok := w.w[kind]; ok {
		return fmt.Errorf("worker for kind %q is already registered", kind)
	}

	w.w[kind] = func(ctx context.Context, args json.RawMessage) error {
		var deserialized T
		err := json.Unmarshal(args, &deserialized)
		if err != nil {
			return fmt.Errorf("failed to deserialize job args: %w", err)
		}

		return worker.Run(ctx, deserialized)
	}

	return nil
}

// MustRegisterJob is like [RegisterJob] but panics on error.
func MustRegisterJob[T Args](w *Workers, worker Job[T]) {
	err := RegisterJob(w, worker)
	if err != nil {
		panic(err)
	}
}

// Submitter returns a submitter instance for this worker.
func (w *Workers) Submitter() *Submitter {
	return &Submitter{w: w}
}

func (w *Workers) runJob(ctx context.Context, dbJob *db.Job) (err error) {
	fn, ok := w.w[dbJob.Kind]
	if !ok {
		return fmt.Errorf("unknown worker kind: %s", dbJob.Kind)
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	err = fn(ctx, json.RawMessage(dbJob.ArgsJson))
	return
}

func (w *Workers) getNextJob(ctx context.Context) (*db.Job, error) {
	job, err := w.d.QueryRW().SetNextJobRunning(ctx, db.SetNextJobRunningParams{
		StartedAt: sqlTimeNow(),
		Pid:       sql.NullInt64{Valid: true, Int64: randomID},
		Now:       time.Now(),
	})
	if err != nil {
		return nil, err
	}

	return &job, nil
}

func (w *Workers) getAndRunAndUpdateNextJob(ctx context.Context) (bool, error) {
	// Retrieve next job.
	job, err := w.getNextJob(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	// Run job.
	logger := logg.GetLogger(ctx).With("jobId", job.ID)
	jobErr := w.runJob(logg.WithLogger(ctx, logger), job)

	// Submit result.
	if jobErr == nil {
		logger.Debug("Job succeeded")
		return true, db.EnsureOneRowChanged(
			w.d.QueryRW().SetJobSuccess(ctx, db.SetJobSuccessParams{
				FinishedAt: sqlTimeNow(),
				ID:         job.ID,
			}))
	}

	next, err := w.d.QueryRW().SetJobError(ctx, db.SetJobErrorParams{
		FinishedAt: sqlTimeNow(),
		Error:      sql.NullString{Valid: true, String: jobErr.Error()},
		ID:         job.ID,
	})
	logger.Error("Job failed", "err", jobErr, "next", next)
	return true, err

}

const waitWhenIdle = time.Second

// RunInBackground spins up the worker goroutines which run jobs.
// Returns immediately.
// Panics if invoked more than once.
func (w *Workers) RunInBackground(ctx context.Context) {
	alreadyRunning := w.running.Swap(true)
	if alreadyRunning {
		logg.Panic(ctx, "Already running")
	}

	nParallel := w.c.MaxParallel
	if nParallel == 0 {
		// #nosec G115 This is fine..
		nParallel = uint(runtime.NumCPU())
	}

	logg.Info(ctx, "Spinning up workers", "n", nParallel)

	for i := uint(0); i < nParallel; i++ {
		go func(ctx context.Context) {
			logger := logg.GetLogger(ctx).With("workerId", i)
			ctx = logg.WithLogger(ctx, logger)
			logg.Info(ctx, "Started")

			for {
				select {
				case <-ctx.Done():
					logg.Info(ctx, "Shutting down", "err", ctx.Err())
					return
				default:
				}

				ranJob, err := w.getAndRunAndUpdateNextJob(ctx)
				if err != nil {
					logg.Error(ctx, "Job runner failure", "err", err)
				}

				if !ranJob {
					logg.Trace(ctx, "Idling")
					time.Sleep(waitWhenIdle)
				}
			}
		}(ctx)
	}
}

// Submitter is a proxy object which can submit jobs to a worker.
type Submitter struct {
	w *Workers
}

// Params are the scheduling parameters for [Submit].
// The zero value is valid and means no delay and no retries.
type Params struct {
	// How many times the job should be retried on failure.
	// 0 means the job will be run once and not retried on failure.
	MaxRetries uint8
	// DelayS is the delay before the job is run.
	// It must be in whole seconds (X * [time.Second]).
	DelayS time.Duration
	// BackofFactorS is the factor applied when calculating the exponential backoff delay:
	//
	// 	factor * (pow(2, retries) - 1)
	//
	// It must be in whole seconds (X * [time.Second]).
	// Exponential backoff can be disabled by setting this value to 0.
	BackofFactorS time.Duration
}

func (c *Params) validate() error {
	if c.DelayS < 0 {
		return errors.New("delay must not be negative")
	}

	if c.DelayS%time.Second != 0 {
		return errors.New("delay must be in whole seconds")
	}

	if c.BackofFactorS < 0 {
		return errors.New("BackofFactorS must not be negative")
	}

	if c.BackofFactorS%time.Second != 0 {
		return errors.New("backoff factor must be in whole seconds")
	}

	return nil
}

// SubmitTx posts a job to the job queue with given args in the given database transaction,
// to be scheduled with the given params.
func SubmitTx[T Args](ctx context.Context, s *Submitter, tx *db.Queries, jobArgs T, params Params) error {
	if err := params.validate(); err != nil {
		return err
	}

	kind := jobArgs.Kind()
	_, ok := s.w.w[kind]
	if !ok {
		return fmt.Errorf("unknown worker kind: %s", kind)
	}

	argsJSON, err := json.Marshal(jobArgs)
	if err != nil {
		return fmt.Errorf("failed to marshal job args: %w", err)
	}

	_, err = tx.CreateJob(ctx, db.CreateJobParams{
		CreatedAt:      time.Now(),
		MaxAttempts:    int64(params.MaxRetries) + 1,
		DelayS:         int64(params.DelayS / time.Second),
		BackoffFactorS: int64(params.BackofFactorS / time.Second),
		Kind:           kind,
		ArgsJson:       string(argsJSON),
	})
	return err
}

// Submit posts a job to the job queue with given args, to be scheduled with the given params.
func Submit[T Args](ctx context.Context, s *Submitter, jobArgs T, params Params) error {
	return SubmitTx(ctx, s, s.w.d.QueryRW(), jobArgs, params)
}

// Periodic schedules a job for periodic submission to the queue.
// Retries and delay cannot be set for periodic jobs and default to 0.
func Periodic[T Args](ctx context.Context, s *Submitter, jobArgs T, period time.Duration) {
	go func() {
		tick := time.NewTicker(period)
		defer tick.Stop()

		logger := logg.GetLogger(ctx).With("kind", jobArgs.Kind(), "period", period, "args", jobArgs)

		for {
			select {
			case <-ctx.Done():
				logger.Info("Shutting down", "err", ctx.Err())
				return
			case <-tick.C:
				err := Submit(ctx, s, jobArgs, Params{})

				if err == nil {
					logger.Debug("Submitted periodic job")
				} else {
					logger.Error("Failed to submit periodic job", "err", err)
				}
			}
		}
	}()
}
