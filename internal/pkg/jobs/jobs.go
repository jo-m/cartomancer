// Package jobs implements an async job queue.
// The db package is used for persistence between restarts.
// Periodically scheduled jobs are also supported.
//
// The interface is partially inspired by https://riverqueue.com/docs.
package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/logg"
	"reflect"
	"runtime"
	"sync/atomic"
	"time"
)

// Args is the interface job args have to implement.
type Args interface {
	// Kind must return a constant, unique string identifying the job type.
	Kind() string
}

// Job is the interface to define custom jobs.
type Job[T Args] interface {
	// Run must be implemented to run the job once.
	// Returning nil means successful execution.
	// Returning an error means that the job will be retried (if maxAttemps is set to > 1).
	Run(ctx context.Context, d *db.DB, args T) error
}

// Config is the configuration for Workers.
type Config struct {
	// MaxParallel is the maximum number of jobs that can run in parallel.
	// It defaults to `runtime.NumCPU()` if zero.
	MaxParallel uint
	// AutoCleanupPeriod is the period at which the built-in cleanup job will run.
	// No cleanup will be performed if set to 0.
	AutoCleanupPeriod time.Duration
	// BackofFactorS is the factor applied when calculating the exponential backoff delay:
	//
	// 	factor * (pow(2, attempts) - 1) [sec]
	//
	// It must be in whole seconds.
	// Exponential backoff can be disabled by setting this value to 0.
	BackofFactorS time.Duration
}

type decodeAndWorkFunc func(ctx context.Context, args json.RawMessage) error

// Workers runs jobs.
// Use NewWorkers() create an instance.
type Workers struct {
	d       *db.DB
	c       Config
	running atomic.Bool
	w       map[string]decodeAndWorkFunc
}

func sqlTimeNow() sql.NullTime {
	return sql.NullTime{Time: time.Now(), Valid: true}
}

// NewWorkers() creates a new workers instance.
// There can only be one instance per process and database.
// Use jobs.RegisterJob() to register new jobs on the worker.
// Use jobs.Submit() and jobs.Periodic() on a submitter (w.Submitter()) to submit jobs to be run.
func NewWorkers(ctx context.Context, d *db.DB, c Config) (*Workers, error) {
	if c.BackofFactorS < 0 {
		return nil, errors.New("backoff factor must not be negative")
	}

	if c.BackofFactorS%time.Second != 0 {
		return nil, errors.New("backoff factor must be in whole seconds")
	}

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
		MustRegisterJob(w, &cleaner{})
		s := w.Submitter()
		Periodic(ctx, s, 1, cleanerArgs{}, c.AutoCleanupPeriod)
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

		return worker.Run(ctx, w.d, deserialized)
	}

	return nil
}

// MustRegisterJob is like RegisterJob() but panics on error.
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
		FactorSec: int64(w.c.BackofFactorS / time.Second),
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
	} else {
		logger.Error("Job failed", "err", jobErr)
		return true, db.EnsureOneRowChanged(
			w.d.QueryRW().SetJobError(ctx, db.SetJobErrorParams{
				FinishedAt: sqlTimeNow(),
				ID:         job.ID,
				Error:      sql.NullString{Valid: true, String: jobErr.Error()},
			}))
	}
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

// Submit posts a job to the job queue with given args,
// to be run with maxAttempts and delay.
// maxAttempts must be at least 1, delay may not be negative.
func Submit[T Args](ctx context.Context, s *Submitter, maxAttempts int, delay time.Duration, jobArgs T) error {
	if maxAttempts < 1 {
		return errors.New("maxAttempts must	be at least 1")
	}

	if delay < 0 {
		return errors.New("delay must not be negative")
	}

	if delay > 0 && delay < time.Second {
		return errors.New("there is no point in specifying a delay smaller than a second")
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

	_, err = s.w.d.QueryRW().CreateJob(ctx, db.CreateJobParams{
		CreatedAt:    time.Now(),
		DelaySeconds: int64(delay.Seconds()),
		MaxAttempts:  int64(maxAttempts),
		Kind:         kind,
		ArgsJson:     string(argsJSON),
	})
	return err
}

// Periodic will periodically post a job to the queue.
// It is dangerous to pass maxAttempts > 1 as the job queue might grow to infinity.
// Exponential backoff is not possible.
func Periodic[T Args](ctx context.Context, s *Submitter, maxAttempts int, jobArgs T, period time.Duration) {
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
				err := Submit(ctx, s, maxAttempts, 0, jobArgs)

				if err == nil {
					logger.Debug("Submitted periodic job")
				} else {
					logger.Error("Failed to submit periodic job", "err", err)
				}
			}
		}
	}()
}
