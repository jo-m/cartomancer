package jobs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"goweb/internal/pkg/db"
	"goweb/internal/pkg/logg"

	"github.com/stretchr/testify/assert"
)

type GoodArgs1 struct {
	Member  string
	Another int
}

func (GoodArgs1) Kind() string {
	return "good1"
}

type GoodWorker1 struct{}

func (m *GoodWorker1) Run(ctx context.Context, args GoodArgs1) error {
	logg.Info(ctx, "Doing good work", "args", args)
	time.Sleep(time.Millisecond * 100)
	return nil
}

type GoodArgs2 struct {
	Member  string
	Another int
}

func (GoodArgs2) Kind() string {
	return "good2"
}

type GoodWorker2 struct{}

func (m *GoodWorker2) Run(ctx context.Context, args GoodArgs2) error {
	logg.Info(ctx, "Doing good work", "args", args)
	time.Sleep(time.Millisecond * 100)
	return nil
}

type BadArgsPrivate struct {
	Member  string
	private int
}

var _ BadArgsPrivate = BadArgsPrivate{private: 1}

func (BadArgsPrivate) Kind() string {
	return "bad"
}

type BadArgsType struct {
	Chan   chan int
	Member complex64
	Mu     *sync.Mutex
}

func (BadArgsType) Kind() string {
	return "bad"
}

func TestCheckArgsType(t *testing.T) {
	assert.NoError(t, checkArgsType[GoodArgs1]())
	assert.Error(t, checkArgsType[BadArgsPrivate]())
	assert.Error(t, checkArgsType[BadArgsType]())
}

func TestUniqueKind(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := logg.WithDiscardHandler(context.Background())
	c := Config{
		MaxParallel:       1,
		AutoCleanupPeriod: 0,
	}

	w, err := NewWorkers(ctx, d, c)
	assert.NoError(t, err)
	assert.NoError(t, RegisterJob(w, &GoodWorker1{}))
	assert.Error(t, RegisterJob(w, &GoodWorker1{}))
	assert.NoError(t, RegisterJob(w, &GoodWorker2{}))
	assert.Error(t, RegisterJob(w, &GoodWorker2{}))
}

func TestUniqueWorkers(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx, cancel := context.WithCancel(context.Background())
	ctx = logg.WithDiscardHandler(ctx)
	c := Config{
		MaxParallel:       1,
		AutoCleanupPeriod: 0,
	}
	defer cancel()

	_, err := NewWorkers(ctx, d, c)
	assert.NoError(t, err)

	_, err = NewWorkers(ctx, d, c)
	assert.ErrorContains(t, err, "only one instance allowed")

	ids, err := d.QueryRO().GetJobRunnerPIDs(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []int64{randomID}, ids)
}

func TestRunnerRunOnlyOnce(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx, cancel := context.WithCancel(context.Background())
	ctx = logg.WithDiscardHandler(ctx)
	c := Config{
		MaxParallel:       1,
		AutoCleanupPeriod: 0,
	}
	defer cancel()

	w, err := NewWorkers(ctx, d, c)
	assert.NoError(t, err)

	w.RunInBackground(ctx)
	assert.Panics(t, func() {
		w.RunInBackground(ctx)
	})
	assert.ErrorContains(t, RegisterJob(w, &GoodWorker1{}), "already running")
}

var cOK chan int = make(chan int, 10000)
var cErr chan int = make(chan int, 10000)

func slurp(c <-chan int, timeout time.Duration) []int {
	var results []int
	timer := time.NewTimer(timeout)

	for {
		select {
		case <-timer.C:
			return results
		case val := <-c:
			results = append(results, val)
		}
	}
}

type TestArgs struct {
	Sleep    time.Duration
	ErrMsg   string
	PanicMsg string
	Val      int
}

func (TestArgs) Kind() string {
	return "chan"
}

type TestWorker struct{}

func (m *TestWorker) Run(ctx context.Context, args TestArgs) error {
	time.Sleep(args.Sleep)

	if args.ErrMsg != "" {
		cErr <- args.Val
		return errors.New(args.ErrMsg)
	}

	if args.PanicMsg != "" {
		panic(args.PanicMsg)
	}

	cOK <- args.Val
	return nil
}

func TestRunJobsParallel(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx, cancel := context.WithCancel(context.Background())
	ctx = logg.WithDiscardHandler(ctx)
	c := Config{
		MaxParallel:       15,
		AutoCleanupPeriod: 0,
	}
	defer cancel()

	w, err := NewWorkers(ctx, d, c)
	assert.NoError(t, err)
	assert.NoError(t, RegisterJob(w, &TestWorker{}))

	s0 := w.Submitter()
	s1 := w.Submitter()
	for i := range 10 {
		args := TestArgs{Val: i, Sleep: time.Millisecond * 100}
		err0 := Submit(ctx, s0, args, Params{})
		err1 := Submit(ctx, s1, args, Params{})
		assert.NoError(t, err0)
		assert.NoError(t, err1)
	}

	// Nothing executed yet.
	results := slurp(cOK, time.Millisecond*150)
	assert.Empty(t, results)

	// Run.
	w.RunInBackground(ctx)

	// We submit 20 jobs taking 100+ms each, and have 15 workers,
	// so we should have 15 jobs done in 100ms, and 5 more in another 100ms.
	results = slurp(cOK, time.Millisecond*120)
	assert.Len(t, results, 15)
	results = slurp(cOK, time.Millisecond*120)
	assert.Len(t, results, 5)
}

func TestRunJobsMaxRetries(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx, cancel := context.WithCancel(context.Background())
	ctx = logg.WithDiscardHandler(ctx)
	c := Config{
		MaxParallel:       15,
		AutoCleanupPeriod: 0,
	}
	defer cancel()

	w, err := NewWorkers(ctx, d, c)
	assert.NoError(t, err)
	assert.NoError(t, RegisterJob(w, &TestWorker{}))

	// Submit 15 failing jobs, each with 2 retries.
	s := w.Submitter()
	for i := range 15 {
		args := TestArgs{Val: i, ErrMsg: "this failed"}
		err := Submit(ctx, s, args, Params{MaxRetries: 2})
		assert.NoError(t, err)
	}
	w.RunInBackground(ctx)

	resultsOK := slurp(cOK, time.Millisecond*100)
	resultsErr := slurp(cErr, time.Millisecond*100)
	assert.Len(t, resultsOK, 0)
	assert.Len(t, resultsErr, 15*3)

	// Check what we have in the db.
	jobs, err := d.QueryRO().GetJobs(ctx)
	assert.NoError(t, err)
	assert.Len(t, jobs, 15)
	for _, j := range jobs {
		assert.Equal(t, int64(3), j.Attempts)
		assert.Equal(t, "this failed", j.Error.String)
	}
}

func TestRunJobsPanicMaxRetries(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx, cancel := context.WithCancel(context.Background())
	ctx = logg.WithDiscardHandler(ctx)
	c := Config{
		MaxParallel:       15,
		AutoCleanupPeriod: 0,
	}
	defer cancel()

	w, err := NewWorkers(ctx, d, c)
	assert.NoError(t, err)
	assert.NoError(t, RegisterJob(w, &TestWorker{}))

	// Submit 15 panicked jobs, each with 3 retries.
	s := w.Submitter()
	for i := range 15 {
		args := TestArgs{Val: i, PanicMsg: "this panicked"}
		err := Submit(ctx, s, args, Params{MaxRetries: 2})
		assert.NoError(t, err)
	}
	w.RunInBackground(ctx)

	resultsOK := slurp(cOK, time.Millisecond*100)
	resultsErr := slurp(cErr, time.Millisecond*100)
	assert.Len(t, resultsOK, 0)
	assert.Len(t, resultsErr, 0)

	// Check what we have in the db.
	jobs, err := d.QueryRO().GetJobs(ctx)
	assert.NoError(t, err)
	assert.Len(t, jobs, 15)
	for _, j := range jobs {
		assert.Equal(t, int64(3), j.Attempts)
		assert.Equal(t, "panic: this panicked", j.Error.String)
	}
}

func TestRunJobsAutoCleanup(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx, cancel := context.WithCancel(context.Background())
	ctx = logg.WithDiscardHandler(ctx)
	c := Config{
		MaxParallel:       30,
		AutoCleanupPeriod: time.Second,
	}
	defer cancel()

	w, err := NewWorkers(ctx, d, c)
	assert.NoError(t, err)
	assert.NoError(t, RegisterJob(w, &TestWorker{}))

	s := w.Submitter()
	for i := range 5 {
		assert.NoError(t, Submit(ctx, s, TestArgs{Val: i}, Params{MaxRetries: 2}))
		assert.NoError(t, Submit(ctx, s, TestArgs{Val: i, ErrMsg: "err msg"}, Params{MaxRetries: 2}))
		assert.NoError(t, Submit(ctx, s, TestArgs{Val: i, PanicMsg: "panic msg"}, Params{MaxRetries: 2}))
	}
	w.RunInBackground(ctx)

	resultsOK := slurp(cOK, time.Millisecond*100)
	resultsErr := slurp(cErr, time.Millisecond*100)
	assert.Len(t, resultsOK, 5)
	assert.Len(t, resultsErr, 15)

	// Time for cleanup.
	time.Sleep(time.Second * 2)

	// Check what we have in the db.
	jobs, err := d.QueryRO().GetJobs(ctx)
	assert.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, jobNameCleaner, jobs[0].Kind)
}

func TestRunJobsPeriodic(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx, cancel := context.WithCancel(context.Background())
	ctx = logg.WithDiscardHandler(ctx)
	c := Config{
		MaxParallel:       15,
		AutoCleanupPeriod: 0,
	}
	defer cancel()

	w, err := NewWorkers(ctx, d, c)
	assert.NoError(t, err)
	assert.NoError(t, RegisterJob(w, &TestWorker{}))

	// Run.
	s := w.Submitter()
	Periodic(ctx, s, TestArgs{Val: 0}, time.Millisecond*10)
	time.Sleep(time.Millisecond * 100)
	w.RunInBackground(ctx)
	time.Sleep(time.Millisecond * 100)
	cancel()

	// Check
	results := slurp(cOK, time.Millisecond*150)
	assert.Len(t, results, 10)
}
