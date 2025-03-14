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

type Args interface {
	Kind() string
}

type Worker[T Args] interface {
	Work(ctx context.Context, d *db.DB, args T) error
}

// Config is the background jobs configuration.
type Config struct {
	// MaxParallel is the maximum number of jobs that can run in parallel.
	// It defaults to `runtime.NumCPU()` if not set.
	MaxParallel uint
	// AutoCleanupPeriod is the period at which the built-in cleanup job will run.
	// No cleanup will be performed if set to 0.
	AutoCleanupPeriod time.Duration
}

type decodeAndWorkFunc func(ctx context.Context, args json.RawMessage) error

type Workers struct {
	d       *db.DB
	c       Config
	running atomic.Bool
	w       map[string]decodeAndWorkFunc
}

func sqlTimeNow() sql.NullTime {
	return sql.NullTime{Time: time.Now(), Valid: true}
}

func NewWorkers(ctx context.Context, d *db.DB, c Config) (*Workers, error) {
	if err := ensureSingleInstance(ctx, d); err != nil {
		return nil, err
	}

	err := d.QueryRW().SetJobsAborted(ctx, db.SetJobsAbortedParams{
		FinishedAt: sqlTimeNow(),
		OurPID:     sql.NullInt64{Valid: true, Int64: randomID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to mark aborted jobs: %w", err)
	}

	w := &Workers{
		d:       d,
		c:       c,
		running: atomic.Bool{},
		w:       map[string]decodeAndWorkFunc{},
	}

	if c.AutoCleanupPeriod != 0 {
		MustAddWorker(w, &cleaner{})
		s := w.Submitter()
		Periodic(ctx, s, 1, cleanupArgs{}, c.AutoCleanupPeriod)
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

func AddWorker[T Args](w *Workers, worker Worker[T]) error {
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

		return worker.Work(ctx, w.d, deserialized)
	}

	return nil
}

func MustAddWorker[T Args](w *Workers, worker Worker[T]) {
	err := AddWorker(w, worker)
	if err != nil {
		panic(err)
	}
}

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
		return true, w.d.QueryRW().SetJobSuccess(ctx, db.SetJobSuccessParams{
			FinishedAt: sqlTimeNow(),
			ID:         job.ID,
		})
	} else {
		return true, w.d.QueryRW().SetJobError(ctx, db.SetJobErrorParams{
			FinishedAt: sqlTimeNow(),
			ID:         job.ID,
			Error:      sql.NullString{Valid: true, String: jobErr.Error()},
		})
	}
}

const waitWhenIdle = time.Second

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

type Submitter struct {
	w *Workers
}

func Submit[T Args](ctx context.Context, s *Submitter, maxAttempts int, jobArgs T) error {
	if maxAttempts < 1 {
		return fmt.Errorf("maxAttempts must	be at least 1")
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
		CreatedAt:   time.Now(),
		MaxAttempts: int64(maxAttempts),
		Kind:        kind,
		ArgsJson:    string(argsJSON),
	})
	return err
}

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
				err := Submit(ctx, s, maxAttempts, jobArgs)

				if err == nil {
					logger.Info("Submitted periodic job")
				} else {
					logger.Error("Failed to submit periodic job", "err", err)
				}
			}
		}
	}()
}
