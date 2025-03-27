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
	"github.com/stretchr/testify/require"
)

// To enable debug logging per test:
// ctx = logg.WithLogger(ctx, slog.New(logg.NewHandler(logg.Config{LogPretty: true, LogLevel: slog.LevelDebug})))

type GoodArgs1 struct {
	Member  string
	Another int
}

func (GoodArgs1) Kind() string {
	return "good1"
}

type GoodJob1 struct{}

func (m *GoodJob1) Run(ctx context.Context, args GoodArgs1) error {
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

type GoodJob2 struct{}

func (m *GoodJob2) Run(ctx context.Context, args GoodArgs2) error {
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
	ctx := logg.WithDiscardHandler(t.Context())
	c := JobsConfig{
		MaxParallel:       1,
		AutoCleanupPeriod: 0,
	}

	w, err := NewWorkers(ctx, d, c)
	assert.NoError(t, err)
	assert.NoError(t, RegisterJob(w, &GoodJob1{}))
	assert.Error(t, RegisterJob(w, &GoodJob1{}))
	assert.NoError(t, RegisterJob(w, &GoodJob2{}))
	assert.Error(t, RegisterJob(w, &GoodJob2{}))
}

func TestUniqueWorkers(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx, cancel := context.WithCancel(t.Context())
	ctx = logg.WithDiscardHandler(ctx)
	c := JobsConfig{
		MaxParallel:       1,
		AutoCleanupPeriod: 0,
	}
	defer cancel()

	_, err := NewWorkers(ctx, d, c)
	require.NoError(t, err)

	_, err = NewWorkers(ctx, d, c)
	assert.ErrorContains(t, err, "only one instance allowed")

	ids, err := d.QueryRO().GetJobRunnerPIDs(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []int64{randomID}, ids)
}

func TestRunnerRunOnlyOnce(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx, cancel := context.WithCancel(t.Context())
	ctx = logg.WithDiscardHandler(ctx)
	c := JobsConfig{
		MaxParallel:       1,
		AutoCleanupPeriod: 0,
	}
	defer cancel()

	w, err := NewWorkers(ctx, d, c)
	require.NoError(t, err)

	w.RunInBackground(ctx)
	assert.Panics(t, func() {
		w.RunInBackground(ctx)
	})
	assert.ErrorContains(t, RegisterJob(w, &GoodJob1{}), "already running")
}

func slurp[T any](c <-chan T, timeout time.Duration) []T {
	var results []T
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

type TestIntArgs struct {
	Sleep    time.Duration
	ErrMsg   string
	PanicMsg string
	Val      int
}

func (TestIntArgs) Kind() string {
	return "jobs.test_int"
}

type TestIntJob struct {
	cOK  chan int
	cErr chan int
}

func newTestIntJob() *TestIntJob {
	return &TestIntJob{
		cOK:  make(chan int, 10000),
		cErr: make(chan int, 10000),
	}
}

func (j *TestIntJob) Run(_ context.Context, args TestIntArgs) error {
	time.Sleep(args.Sleep)

	if args.ErrMsg != "" {
		j.cErr <- args.Val
		return errors.New(args.ErrMsg)
	}

	if args.PanicMsg != "" {
		panic(args.PanicMsg)
	}

	j.cOK <- args.Val
	return nil
}

func TestRunJobsParallel(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx, cancel := context.WithCancel(t.Context())
	ctx = logg.WithDiscardHandler(ctx)
	c := JobsConfig{
		MaxParallel:       15,
		AutoCleanupPeriod: 0,
	}
	defer cancel()

	w, err := NewWorkers(ctx, d, c)
	require.NoError(t, err)
	j := newTestIntJob()
	require.NoError(t, RegisterJob(w, j))

	s0 := w.Submitter()
	s1 := w.Submitter()
	for i := range 10 {
		args := TestIntArgs{Val: i, Sleep: time.Millisecond * 100}
		err0 := Submit(ctx, s0, args, Params{})
		err1 := Submit(ctx, s1, args, Params{})
		require.NoError(t, err0)
		require.NoError(t, err1)
	}

	// Nothing executed yet.
	results := slurp(j.cOK, time.Millisecond*150)
	assert.Empty(t, results)

	// Run.
	w.RunInBackground(ctx)

	// We submit 20 jobs taking 100+ms each, and have 15 workers,
	// so we should have 15 jobs done in 100ms, and 5 more in another 100ms.
	results = slurp(j.cOK, time.Millisecond*120)
	assert.Len(t, results, 15)
	results = slurp(j.cOK, time.Millisecond*120)
	assert.Len(t, results, 5)
}

func TestRunJobsMaxRetries(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx, cancel := context.WithCancel(t.Context())
	ctx = logg.WithDiscardHandler(ctx)
	c := JobsConfig{
		MaxParallel:       15,
		AutoCleanupPeriod: 0,
	}
	defer cancel()

	w, err := NewWorkers(ctx, d, c)
	require.NoError(t, err)
	j := newTestIntJob()
	require.NoError(t, RegisterJob(w, j))

	// Submit 15 failing jobs, each with 2 retries.
	s := w.Submitter()
	for i := range 15 {
		args := TestIntArgs{Val: i, ErrMsg: "this failed"}
		err := Submit(ctx, s, args, Params{MaxRetries: 2})
		require.NoError(t, err)
	}
	w.RunInBackground(ctx)

	resultsOK := slurp(j.cOK, time.Millisecond*100)
	resultsErr := slurp(j.cErr, time.Millisecond*100)
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
	ctx, cancel := context.WithCancel(t.Context())
	ctx = logg.WithDiscardHandler(ctx)
	c := JobsConfig{
		MaxParallel:       15,
		AutoCleanupPeriod: 0,
	}
	defer cancel()

	w, err := NewWorkers(ctx, d, c)
	require.NoError(t, err)
	j := newTestIntJob()
	require.NoError(t, RegisterJob(w, j))

	// Submit 15 panicked jobs, each with 3 retries.
	s := w.Submitter()
	for i := range 15 {
		args := TestIntArgs{Val: i, PanicMsg: "this panicked"}
		err := Submit(ctx, s, args, Params{MaxRetries: 2})
		require.NoError(t, err)
	}
	w.RunInBackground(ctx)

	resultsOK := slurp(j.cOK, time.Millisecond*100)
	resultsErr := slurp(j.cErr, time.Millisecond*100)
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
	ctx, cancel := context.WithCancel(t.Context())
	ctx = logg.WithDiscardHandler(ctx)
	c := JobsConfig{
		MaxParallel:       30,
		AutoCleanupPeriod: time.Second,
	}
	defer cancel()

	w, err := NewWorkers(ctx, d, c)
	require.NoError(t, err)
	j := newTestIntJob()
	require.NoError(t, RegisterJob(w, j))

	s := w.Submitter()
	for i := range 5 {
		require.NoError(t, Submit(ctx, s, TestIntArgs{Val: i}, Params{MaxRetries: 2}))
		require.NoError(t, Submit(ctx, s, TestIntArgs{Val: i, ErrMsg: "err msg"}, Params{MaxRetries: 2}))
		require.NoError(t, Submit(ctx, s, TestIntArgs{Val: i, PanicMsg: "panic msg"}, Params{MaxRetries: 2}))
	}
	w.RunInBackground(ctx)

	resultsOK := slurp(j.cOK, time.Millisecond*100)
	resultsErr := slurp(j.cErr, time.Millisecond*100)
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
	ctx, cancel := context.WithCancel(t.Context())
	ctx = logg.WithDiscardHandler(ctx)
	c := JobsConfig{
		MaxParallel:       15,
		AutoCleanupPeriod: 0,
	}
	defer cancel()

	w, err := NewWorkers(ctx, d, c)
	require.NoError(t, err)
	j := newTestIntJob()
	require.NoError(t, RegisterJob(w, j))

	// Run.
	s := w.Submitter()
	Periodic(ctx, s, TestIntArgs{Val: 0}, time.Millisecond*10)
	time.Sleep(time.Millisecond * 100)
	w.RunInBackground(ctx)
	time.Sleep(time.Millisecond * 100)
	cancel()

	// Check
	results := slurp(j.cOK, time.Millisecond*150)
	assert.Len(t, results, 10)
}

type TestTimeArgs struct {
	T0   time.Time
	Fail string
}

func (TestTimeArgs) Kind() string {
	return "jobs.test_time"
}

type TestTimeJob struct {
	c chan time.Duration
}

func newTestTimeJob() *TestTimeJob {
	return &TestTimeJob{
		c: make(chan time.Duration, 10000),
	}
}

func (j *TestTimeJob) Run(_ context.Context, args TestTimeArgs) error {
	j.c <- time.Since(args.T0)
	if args.Fail != "" {
		return errors.New(args.Fail)
	}
	return nil
}

func durationsToS(d []time.Duration) []int {
	ret := make([]int, len(d))
	for i := range d {
		ret[i] = int(d[i] / time.Second)
	}
	return ret
}

func TestRunJobsDelay(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx, cancel := context.WithCancel(t.Context())
	ctx = logg.WithDiscardHandler(ctx)
	c := JobsConfig{
		MaxParallel: 2,
	}
	defer cancel()

	w, err := NewWorkers(ctx, d, c)
	require.NoError(t, err)
	j := newTestTimeJob()
	require.NoError(t, RegisterJob(w, j))

	s := w.Submitter()
	require.NoError(t, Submit(ctx, s, TestTimeArgs{T0: time.Now()}, Params{DelayS: time.Second * 0}))
	require.NoError(t, Submit(ctx, s, TestTimeArgs{T0: time.Now()}, Params{DelayS: time.Second * 1}))
	require.NoError(t, Submit(ctx, s, TestTimeArgs{T0: time.Now()}, Params{DelayS: time.Second * 2}))
	w.RunInBackground(ctx)

	delays := slurp(j.c, time.Second*4)
	delaysS := durationsToS(delays)
	assert.Equal(t, []int{0, 1, 2}, delaysS)
}

func TestRunJobsBackoff(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx, cancel := context.WithCancel(t.Context())
	ctx = logg.WithDiscardHandler(ctx)
	c := JobsConfig{
		MaxParallel: 2,
	}
	defer cancel()

	w, err := NewWorkers(ctx, d, c)
	require.NoError(t, err)
	j := newTestTimeJob()
	require.NoError(t, RegisterJob(w, j))

	s := w.Submitter()
	args := TestTimeArgs{T0: time.Now(), Fail: "failed"}
	params := Params{MaxRetries: 3, BackofFactorS: time.Second * 1}
	require.NoError(t, Submit(ctx, s, args, params))
	w.RunInBackground(ctx)

	delays := slurp(j.c, time.Second*8)
	delaysS := durationsToS(delays)
	assert.Equal(t, []int{0, 1, 3, 7}, delaysS)
}
