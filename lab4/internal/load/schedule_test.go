package load

import (
	"testing"
	"time"
)

func TestScheduleInterpolatesBetweenStages(t *testing.T) {
	schedule := Schedule{
		{Duration: 10 * time.Second, Target: 100},
		{Duration: 10 * time.Second, Target: 100},
		{Duration: 10 * time.Second, Target: 0},
	}

	cases := map[time.Duration]int{
		0:                0,
		5 * time.Second:  50,
		10 * time.Second: 100,
		15 * time.Second: 100,
		25 * time.Second: 50,
		40 * time.Second: 0,
	}
	for elapsed, want := range cases {
		if got := schedule.Target(elapsed); got != want {
			t.Errorf("Target(%s) = %d, want %d", elapsed, got, want)
		}
	}
	if total := schedule.Total(); total != 30*time.Second {
		t.Errorf("Total() = %s, want 30s", total)
	}
}

func TestConstantScenarioStartsAtFullLoad(t *testing.T) {
	plan := &Scenario{Kind: Constant, VUs: 300, DurationS: 600, WarmupS: 60}
	schedule := Schedule(plan.Stages(1))

	if got := schedule.Target(0); got != 300 {
		t.Errorf("Target(0) = %d, want 300", got)
	}
	if total := schedule.Total(); total != 600*time.Second {
		t.Errorf("Total() = %s, want 10m", total)
	}
}

func TestConstantScenarioRampsUpGradually(t *testing.T) {
	plan := &Scenario{Kind: Constant, VUs: 400, DurationS: 300, RampS: 60, WarmupS: 30}
	schedule := Schedule(plan.Stages(1))

	cases := map[time.Duration]int{
		0:                 0,
		30 * time.Second:  200,
		60 * time.Second:  400,
		200 * time.Second: 400,
	}
	for elapsed, want := range cases {
		if got := schedule.Target(elapsed); got != want {
			t.Errorf("Target(%s) = %d, want %d", elapsed, got, want)
		}
	}
	if total := schedule.Total(); total != 300*time.Second {
		t.Errorf("Total() = %s, want 5m", total)
	}
}
