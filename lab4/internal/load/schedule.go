package load

import (
	"math"
	"time"
)

type Schedule []Stage

func (s Schedule) Total() time.Duration {
	var total time.Duration
	for _, stage := range s {
		total += stage.Duration
	}
	return total
}

func (s Schedule) Target(elapsed time.Duration) int {
	from := 0
	for _, stage := range s {
		if elapsed < stage.Duration {
			progress := float64(elapsed) / float64(stage.Duration)
			return int(math.Round(float64(from) + progress*float64(stage.Target-from)))
		}
		elapsed -= stage.Duration
		from = stage.Target
	}
	return from
}
