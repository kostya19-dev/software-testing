package metrics

import "fmt"

// Limits are the upper bounds a function must stay within. Zero disables a limit.
type Limits struct {
	Cyclomatic int     `json:"max_cyclomatic"`
	Volume     float64 `json:"max_halstead_volume"`
	Difficulty float64 `json:"max_halstead_difficulty"`
	Effort     float64 `json:"max_halstead_effort"`
}

// Violation is a single exceeded limit.
type Violation struct {
	Function Function `json:"function"`
	Metric   string   `json:"metric"`
	Value    float64  `json:"value"`
	Limit    float64  `json:"limit"`
}

func (v Violation) String() string {
	return fmt.Sprintf("%s: %s %s = %.1f > %.0f", v.Function.Location(), v.Function.Name, v.Metric, v.Value, v.Limit)
}

// Check returns every limit exceeded by the given functions.
func Check(functions []Function, limits Limits) []Violation {
	var violations []Violation
	for _, f := range functions {
		for _, c := range []struct {
			metric       string
			value, limit float64
		}{
			{"cyclomatic complexity", float64(f.Cyclomatic), float64(limits.Cyclomatic)},
			{"Halstead volume", f.Halstead.Volume, limits.Volume},
			{"Halstead difficulty", f.Halstead.Difficulty, limits.Difficulty},
			{"Halstead effort", f.Halstead.Effort, limits.Effort},
		} {
			if c.limit > 0 && c.value > c.limit {
				violations = append(violations, Violation{Function: f, Metric: c.metric, Value: c.value, Limit: c.limit})
			}
		}
	}
	return violations
}

// Summary aggregates metrics over all analysed functions.
type Summary struct {
	Functions          int     `json:"functions"`
	MaxCyclomatic      int     `json:"max_cyclomatic"`
	MeanCyclomatic     float64 `json:"mean_cyclomatic"`
	MaxVolume          float64 `json:"max_volume"`
	TotalVolume        float64 `json:"total_volume"`
	MaxDifficulty      float64 `json:"max_difficulty"`
	TotalEffort        float64 `json:"total_effort"`
	EstimatedBugs      float64 `json:"estimated_bugs"`
	MinMaintainability float64 `json:"min_maintainability"`
}

// Summarize computes aggregate figures.
func Summarize(functions []Function) Summary {
	s := Summary{Functions: len(functions), MinMaintainability: 100}
	cyclomatic := 0
	for _, f := range functions {
		cyclomatic += f.Cyclomatic
		s.MaxCyclomatic = max(s.MaxCyclomatic, f.Cyclomatic)
		s.MaxVolume = max(s.MaxVolume, f.Halstead.Volume)
		s.MaxDifficulty = max(s.MaxDifficulty, f.Halstead.Difficulty)
		s.MinMaintainability = min(s.MinMaintainability, f.Maintainability)
		s.TotalVolume += f.Halstead.Volume
		s.TotalEffort += f.Halstead.Effort
		s.EstimatedBugs += f.Halstead.Bugs
	}
	if len(functions) > 0 {
		s.MeanCyclomatic = float64(cyclomatic) / float64(len(functions))
	}
	return s
}
