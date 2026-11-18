package report

import (
	"fmt"
	"slices"
	"strings"
)

func verdict(s Summary) string {
	switch {
	case s.Capacity != nil:
		c := s.Capacity
		stable := "не достигнута"
		if c.StableVUs > 0 {
			stable = fmt.Sprintf("%d VU (~%s RPS)", c.StableVUs, formatNumber(c.StableRPS))
		}
		breaking := "не найдена в пределах теста"
		if c.BreakingVU > 0 {
			breaking = fmt.Sprintf("%d VU", c.BreakingVU)
		}
		return fmt.Sprintf("Стабильная работа до %s. Нарушение SLO начинается с %s. Максимальная пропускная способность %s RPS при %d VU.",
			stable, breaking, formatNumber(c.PeakRPS), c.PeakRPSVUs)
	case s.Recovery != nil:
		r := s.Recovery
		recovery := "не произошло до конца теста"
		if r.Recovered {
			recovery = fmt.Sprintf("за %s с", formatNumber(r.RecoveryTimeS))
		}
		return fmt.Sprintf("Базовая нагрузка: p95 %s мс, %s RPS. Перегрузка: p95 %s мс, отказы %s, %s RPS. Восстановление после снятия перегрузки %s.",
			formatNumber(r.Baseline.P95), formatNumber(r.Baseline.RPSMean),
			formatNumber(r.Overload.P95), formatPercent(float64(r.Overload.FailureRate), 1), formatNumber(r.Overload.RPSMean), recovery)
	default:
		st := s.Stability
		components := make([]string, 0, len(st.MemoryGrowthMBPerH))
		for name := range st.MemoryGrowthMBPerH {
			components = append(components, name)
		}
		slices.Sort(components)
		growth := make([]string, len(components))
		for i, name := range components {
			growth[i] = fmt.Sprintf("%s %s МБ/ч", name, formatNumber(st.MemoryGrowthMBPerH[name]))
		}
		return fmt.Sprintf("Ошибочных операций: задано %s, фактически %s. Отказы %s. Дрейф p95 между первым и последним отрезком ×%s, RPS ×%s. Тренд памяти: %s.",
			formatPercent(st.ConfiguredErrorRate, 0), formatPercent(float64(s.Overall.InvalidShare), 1), formatPercent(float64(s.Overall.FailureRate), 2),
			formatNumber(st.P95Drift), formatNumber(st.RPSDrift), strings.Join(growth, ", "))
	}
}

func headline(s Summary) string {
	switch {
	case s.Capacity != nil && s.Capacity.BreakingVU == 0:
		return fmt.Sprintf("SLO выполнено на всех ступенях до %d VU, пик %s RPS", s.Capacity.StableVUs, formatNumber(s.Capacity.PeakRPS))
	case s.Capacity != nil:
		return fmt.Sprintf("стабильно до %d VU, нарушение SLO с %d VU, пик %s RPS", s.Capacity.StableVUs, s.Capacity.BreakingVU, formatNumber(s.Capacity.PeakRPS))
	case s.Recovery != nil:
		if s.Recovery.Recovered {
			return fmt.Sprintf("восстановление за %s с", formatNumber(s.Recovery.RecoveryTimeS))
		}
		return "не восстановилась до конца теста"
	default:
		return fmt.Sprintf("дрейф p95 ×%s, ошибочных %s", formatNumber(s.Stability.P95Drift), formatPercent(float64(s.Overall.InvalidShare), 1))
	}
}
