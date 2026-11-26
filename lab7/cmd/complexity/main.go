// Command complexity reports cyclomatic and Halstead complexity of Go code
// and fails when a function exceeds the configured limits.
//
//	complexity -dir ../project/VitaLineClinic -max-cyclomatic 10 -max-volume 3000 -max-difficulty 30
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"vitaline-lab7/internal/metrics"
)

type report struct {
	Dir        string              `json:"dir"`
	Limits     metrics.Limits      `json:"limits"`
	Summary    metrics.Summary     `json:"summary"`
	Violations []metrics.Violation `json:"violations"`
	Functions  []metrics.Function  `json:"functions"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("complexity", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "каталог с Go-кодом")
	tests := fs.Bool("tests", true, "анализировать *_test.go")
	exclude := fs.String("exclude", "", "фрагменты путей через запятую, которые не анализируются")
	top := fs.Int("top", 10, "сколько самых сложных функций показать")
	jsonPath := fs.String("json", "", "сохранить полный отчёт в JSON")
	mdPath := fs.String("markdown", "", "дописать сводку в Markdown (например, $GITHUB_STEP_SUMMARY)")
	var limits metrics.Limits
	fs.IntVar(&limits.Cyclomatic, "max-cyclomatic", 10, "максимальная цикломатическая сложность функции (0 — без ограничения)")
	fs.Float64Var(&limits.Volume, "max-volume", 3000, "максимальный объём Холстеда V функции")
	fs.Float64Var(&limits.Difficulty, "max-difficulty", 30, "максимальная сложность Холстеда D функции")
	fs.Float64Var(&limits.Effort, "max-effort", 0, "максимальные усилия Холстеда E функции")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	functions, err := metrics.AnalyzeDir(*dir, metrics.Options{IncludeTests: *tests, Exclude: splitList(*exclude)})
	if err != nil {
		fmt.Fprintln(stderr, "complexity:", err)
		return 2
	}
	r := report{Dir: *dir, Limits: limits, Summary: metrics.Summarize(functions), Functions: functions}
	r.Violations = metrics.Check(functions, limits)

	printText(stdout, r, *top)
	if err := writeOutputs(r, *jsonPath, *mdPath, *top); err != nil {
		fmt.Fprintln(stderr, "complexity:", err)
		return 2
	}
	if len(r.Violations) > 0 {
		return 1
	}
	return 0
}

func splitList(value string) []string {
	var result []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

func writeOutputs(r report, jsonPath, mdPath string, top int) error {
	if jsonPath != "" {
		raw, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(jsonPath), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(jsonPath, raw, 0o600); err != nil {
			return err
		}
	}
	if mdPath != "" {
		f, err := os.OpenFile(mdPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		printMarkdown(f, r, top)
		return f.Close()
	}
	return nil
}

func byVolume(functions []metrics.Function, top int) []metrics.Function {
	sorted := append([]metrics.Function(nil), functions...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Halstead.Volume > sorted[j].Halstead.Volume })
	return sorted[:min(top, len(sorted))]
}

func printText(w io.Writer, r report, top int) {
	s := r.Summary
	fmt.Fprintf(w, "Функций: %d; цикломатическая сложность: средняя %.2f, макс. %d (лимит %d)\n",
		s.Functions, s.MeanCyclomatic, s.MaxCyclomatic, r.Limits.Cyclomatic)
	fmt.Fprintf(w, "Холстед: макс. V %.0f (лимит %.0f), макс. D %.1f (лимит %.0f), сумма V %.0f, E %.0f, ожидаемых ошибок B %.2f; мин. индекс сопровождаемости %.1f\n\n",
		s.MaxVolume, r.Limits.Volume, s.MaxDifficulty, r.Limits.Difficulty, s.TotalVolume, s.TotalEffort, s.EstimatedBugs, s.MinMaintainability)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "CC\tn1\tn2\tN1\tN2\tV\tD\tE\tB\tMI\tФункция\tМесто")
	for _, f := range byVolume(r.Functions, top) {
		h := f.Halstead
		fmt.Fprintf(tw, "%d\t%d\t%d\t%d\t%d\t%.0f\t%.1f\t%.0f\t%.3f\t%.1f\t%s\t%s\n",
			f.Cyclomatic, h.DistinctOperators, h.DistinctOperands, h.TotalOperators, h.TotalOperands,
			h.Volume, h.Difficulty, h.Effort, h.Bugs, f.Maintainability, f.Name, f.Location())
	}
	_ = tw.Flush()

	if len(r.Violations) == 0 {
		fmt.Fprintln(w, "\nOK: все функции в пределах лимитов")
		return
	}
	fmt.Fprintf(w, "\nПревышены лимиты (%d):\n", len(r.Violations))
	for _, v := range r.Violations {
		fmt.Fprintln(w, "  "+v.String())
	}
}

func printMarkdown(w io.Writer, r report, top int) {
	s := r.Summary
	fmt.Fprintf(w, "### Метрики сложности\n\n")
	fmt.Fprintf(w, "| Метрика | Значение | Лимит |\n| --- | ---: | ---: |\n")
	fmt.Fprintf(w, "| Функций | %d | — |\n", s.Functions)
	fmt.Fprintf(w, "| Цикломатическая сложность, макс. / средняя | %d / %.2f | %d |\n", s.MaxCyclomatic, s.MeanCyclomatic, r.Limits.Cyclomatic)
	fmt.Fprintf(w, "| Объём Холстеда V, макс. | %.0f | %.0f |\n", s.MaxVolume, r.Limits.Volume)
	fmt.Fprintf(w, "| Сложность Холстеда D, макс. | %.1f | %.0f |\n", s.MaxDifficulty, r.Limits.Difficulty)
	fmt.Fprintf(w, "| Ожидаемое число ошибок B, сумма | %.2f | — |\n", s.EstimatedBugs)
	fmt.Fprintf(w, "| Индекс сопровождаемости, мин. | %.1f | — |\n\n", s.MinMaintainability)
	fmt.Fprintf(w, "| Функция | CC | V | D | E |\n| --- | ---: | ---: | ---: | ---: |\n")
	for _, f := range byVolume(r.Functions, top) {
		fmt.Fprintf(w, "| `%s` (%s) | %d | %.0f | %.1f | %.0f |\n", f.Name, f.Location(), f.Cyclomatic, f.Halstead.Volume, f.Halstead.Difficulty, f.Halstead.Effort)
	}
	fmt.Fprintf(w, "\nПревышений лимитов: %d\n", len(r.Violations))
	for _, v := range r.Violations {
		fmt.Fprintf(w, "- %s\n", v)
	}
	fmt.Fprintln(w)
}
