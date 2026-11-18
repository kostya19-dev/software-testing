package load

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"time"
)

type Kind string

const (
	Steps    Kind = "steps"
	Spike    Kind = "spike"
	Constant Kind = "constant"
)

type Slo struct {
	P95Ms       float64 `json:"p95_ms"`
	FailureRate float64 `json:"failure_rate"`
}

type Scenario struct {
	Name        string  `json:"-"`
	Title       string  `json:"title"`
	Kind        Kind    `json:"kind"`
	Levels      []int   `json:"levels,omitempty"`
	RampS       float64 `json:"ramp_s,omitempty"`
	HoldS       float64 `json:"hold_s,omitempty"`
	BaselineVUs int     `json:"baseline_vus,omitempty"`
	PeakVUs     int     `json:"peak_vus,omitempty"`
	WarmupS     float64 `json:"warmup_s,omitempty"`
	OverloadS   float64 `json:"overload_s,omitempty"`
	RecoveryS   float64 `json:"recovery_s,omitempty"`
	VUs         int     `json:"vus,omitempty"`
	DurationS   float64 `json:"duration_s,omitempty"`
	ErrorRate   float64 `json:"error_rate"`
	Slo         Slo     `json:"-"`
}

type Catalog struct {
	Slo       Slo                  `json:"slo"`
	Scenarios map[string]*Scenario `json:"scenarios"`
}

func LoadCatalog(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var catalog Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	for name, s := range catalog.Scenarios {
		s.Name = name
		s.Slo = catalog.Slo
	}
	return &catalog, nil
}

func (c *Catalog) Get(name string) (*Scenario, error) {
	s, ok := c.Scenarios[name]
	if !ok {
		return nil, fmt.Errorf("unknown scenario %q, expected one of %v", name, c.Names())
	}
	return s, nil
}

func (c *Catalog) Names() []string {
	order := map[Kind]int{Steps: 0, Spike: 1, Constant: 2}
	names := make([]string, 0, len(c.Scenarios))
	for name := range c.Scenarios {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		a, b := c.Scenarios[names[i]], c.Scenarios[names[j]]
		if order[a.Kind] != order[b.Kind] {
			return order[a.Kind] < order[b.Kind]
		}
		return names[i] < names[j]
	})
	return names
}

type Phase struct {
	Name   string  `json:"name"`
	Start  float64 `json:"start_s"`
	End    float64 `json:"end_s"`
	VUs    int     `json:"vus"`
	Steady bool    `json:"steady"`
}

func (p Phase) Duration() float64 {
	return p.End - p.Start
}

type Stage struct {
	Duration time.Duration
	Target   int
}

func (s *Scenario) Phases(scale float64) []Phase {
	b := phaseBuilder{scale: scale}
	switch s.Kind {
	case Steps:
		for _, level := range s.Levels {
			b.add(fmt.Sprintf("разгон до %d", level), s.RampS, level, false)
			b.add(fmt.Sprintf("%d VU", level), s.HoldS, level, true)
		}
	case Spike:
		b.add("разгон", s.RampS, s.BaselineVUs, false)
		b.add("базовая нагрузка", s.WarmupS, s.BaselineVUs, true)
		b.add("рост", s.RampS, s.PeakVUs, false)
		b.add("перегрузка", s.OverloadS, s.PeakVUs, true)
		b.add("спад", s.RampS, s.BaselineVUs, false)
		b.add("восстановление", s.RecoveryS, s.BaselineVUs, true)
	case Constant:
		ramp := math.Min(s.RampS, s.DurationS)
		warmup := math.Min(s.WarmupS, s.DurationS-ramp)
		if ramp > 0 {
			b.add("разгон", ramp, s.VUs, false)
		}
		b.add("прогрев", warmup, s.VUs, false)
		b.add("стабильная нагрузка", s.DurationS-ramp-warmup, s.VUs, true)
	}
	return b.phases
}

func (s *Scenario) Stages(scale float64) []Stage {
	phases := s.Phases(scale)
	stages := make([]Stage, 0, len(phases))
	for i, phase := range phases {
		duration := time.Duration(phase.Duration() * float64(time.Second))
		if s.Kind == Constant && s.RampS == 0 && i == 0 {
			stages = append(stages, Stage{Duration: 0, Target: phase.VUs})
		}
		stages = append(stages, Stage{Duration: duration, Target: phase.VUs})
	}
	return stages
}

type phaseBuilder struct {
	scale  float64
	cursor float64
	phases []Phase
}

func (b *phaseBuilder) add(name string, seconds float64, vus int, steady bool) {
	length := ScaledSeconds(seconds, b.scale)
	b.phases = append(b.phases, Phase{Name: name, Start: b.cursor, End: b.cursor + length, VUs: vus, Steady: steady})
	b.cursor += length
}

func ScaledSeconds(value, scale float64) float64 {
	return math.Max(1, math.Floor(value*scale+0.5))
}
