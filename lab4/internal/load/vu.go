package load

import (
	"context"
	"math/rand/v2"
	"time"
)

type VU struct {
	ID   int
	ctx  context.Context
	stop <-chan struct{}
}

func (v *VU) Context() context.Context {
	return v.ctx
}

func (v *VU) Running() bool {
	select {
	case <-v.stop:
		return false
	case <-v.ctx.Done():
		return false
	default:
		return true
	}
}

func (v *VU) Pause(min, max time.Duration) bool {
	timer := time.NewTimer(min + rand.N(max-min))
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-v.stop:
		return false
	case <-v.ctx.Done():
		return false
	}
}

func (v *VU) Chance(probability float64) bool {
	return rand.Float64() < probability
}
