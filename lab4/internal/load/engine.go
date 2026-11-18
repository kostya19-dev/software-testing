package load

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

const controlInterval = 100 * time.Millisecond

type Session interface {
	Iterate(vu *VU)
}

type Engine struct {
	Schedule   Schedule
	Grace      time.Duration
	NewSession func(id int) Session

	active   atomic.Int64
	sessions map[int]*seat
	workers  []*worker
	wg       sync.WaitGroup
}

type seat struct {
	session  Session
	released chan struct{}
}

type worker struct {
	vu   *VU
	stop chan struct{}
}

func (e *Engine) Active() int {
	return int(e.active.Load())
}

func (e *Engine) Run(ctx context.Context) {
	total := e.Schedule.Total()
	hard, cancel := context.WithTimeout(ctx, total+e.Grace)
	defer cancel()

	e.sessions = make(map[int]*seat)
	start := time.Now()
	ticker := time.NewTicker(controlInterval)
	defer ticker.Stop()

	for elapsed := time.Duration(0); elapsed < total && ctx.Err() == nil; elapsed = time.Since(start) {
		e.scale(hard, e.Schedule.Target(elapsed))
		select {
		case <-ticker.C:
		case <-ctx.Done():
		}
	}
	e.scale(hard, 0)
	e.wg.Wait()
}

func (e *Engine) scale(ctx context.Context, target int) {
	for len(e.workers) < target {
		e.spawn(ctx, len(e.workers)+1)
	}
	for len(e.workers) > target {
		last := e.workers[len(e.workers)-1]
		close(last.stop)
		e.workers = e.workers[:len(e.workers)-1]
	}
}

func (e *Engine) spawn(ctx context.Context, id int) {
	previous, ok := e.sessions[id]
	if !ok {
		previous = &seat{session: e.NewSession(id), released: closedChannel()}
	}
	current := &seat{session: previous.session, released: make(chan struct{})}
	e.sessions[id] = current

	w := &worker{stop: make(chan struct{})}
	w.vu = &VU{ID: id, ctx: ctx, stop: w.stop}
	e.workers = append(e.workers, w)

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		defer close(current.released)
		<-previous.released
		e.active.Add(1)
		defer e.active.Add(-1)
		for w.vu.Running() {
			current.session.Iterate(w.vu)
		}
	}()
}

func closedChannel() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
