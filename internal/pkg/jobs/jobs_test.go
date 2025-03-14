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

func (m *GoodWorker1) Work(ctx context.Context, _ *db.DB, args GoodArgs1) error {
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

func (m *GoodWorker2) Work(ctx context.Context, _ *db.DB, args GoodArgs2) error {
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
	assert.NoError(t, AddWorker(w, &GoodWorker1{}))
	assert.Error(t, AddWorker(w, &GoodWorker1{}))
	assert.NoError(t, AddWorker(w, &GoodWorker2{}))
	assert.Error(t, AddWorker(w, &GoodWorker2{}))
}

func TestUniqueRunner(t *testing.T) {
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

	r, err := w.Runner(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, r)

	_, err = w.Runner(ctx)
	assert.ErrorContains(t, err, "there is already another workers instance")
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

	r, err := w.Runner(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, r)

	r.RunInBackground(ctx)
	assert.Panics(t, func() {
		r.RunInBackground(ctx)
	})
	assert.ErrorContains(t, AddWorker(w, &GoodWorker1{}), "already running")
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

func (m *TestWorker) Work(ctx context.Context, _ *db.DB, args TestArgs) error {
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
	assert.NoError(t, AddWorker(w, &TestWorker{}))

	r, err := w.Runner(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, r)

	s0 := w.Submitter()
	s1 := w.Submitter()
	for i := range 10 {
		args := TestArgs{Val: i, Sleep: time.Millisecond * 100}
		err0 := Submit(ctx, s0, 1, args)
		err1 := Submit(ctx, s1, 1, args)
		assert.NoError(t, err0)
		assert.NoError(t, err1)
	}

	// Nothing executed yet.
	results := slurp(cOK, time.Millisecond*150)
	assert.Empty(t, results)

	// Run.
	r.RunInBackground(ctx)

	// We submit 20 jobs taking 100+ms each, and have 15 workers,
	// so we should have 15 jobs done in 100ms, and 5 more in another 100ms.
	results = slurp(cOK, time.Millisecond*105)
	assert.Len(t, results, 15)
	results = slurp(cOK, time.Millisecond*105)
	assert.Len(t, results, 5)
}

func TestRunJobsMaxAttempts(t *testing.T) {
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
	assert.NoError(t, AddWorker(w, &TestWorker{}))

	r, err := w.Runner(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, r)

	// Submit 15 failing jobs, each with 3 attempts
	s := w.Submitter()
	for i := range 15 {
		args := TestArgs{Val: i, ErrMsg: "this failed"}
		err := Submit(ctx, s, 3, args)
		assert.NoError(t, err)
	}
	r.RunInBackground(ctx)

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

func TestRunJobsPanicMaxAttempts(t *testing.T) {
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
	assert.NoError(t, AddWorker(w, &TestWorker{}))

	r, err := w.Runner(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, r)

	// Submit 15 panicing jobs, each with 3 attempts
	s := w.Submitter()
	for i := range 15 {
		args := TestArgs{Val: i, PanicMsg: "this paniced"}
		err := Submit(ctx, s, 3, args)
		assert.NoError(t, err)
	}
	r.RunInBackground(ctx)

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
		assert.Equal(t, "panic: this paniced", j.Error.String)
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
	assert.NoError(t, AddWorker(w, &TestWorker{}))

	r, err := w.Runner(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, r)

	s := w.Submitter()
	for i := range 5 {
		assert.NoError(t, Submit(ctx, s, 3, TestArgs{Val: i}))
		assert.NoError(t, Submit(ctx, s, 3, TestArgs{Val: i, ErrMsg: "err msg"}))
		assert.NoError(t, Submit(ctx, s, 3, TestArgs{Val: i, PanicMsg: "panic msg"}))
	}
	r.RunInBackground(ctx)

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
	assert.Equal(t, jobNameCleanup, jobs[0].Kind)
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
	assert.NoError(t, AddWorker(w, &TestWorker{}))

	r, err := w.Runner(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, r)

	// Run.
	s := w.Submitter()
	Periodic(ctx, s, 1, TestArgs{Val: 0}, time.Millisecond*10)
	time.Sleep(time.Millisecond * 100)
	r.RunInBackground(ctx)
	time.Sleep(time.Millisecond * 100)
	cancel()

	// Check
	results := slurp(cOK, time.Millisecond*150)
	assert.Len(t, results, 10)
}
