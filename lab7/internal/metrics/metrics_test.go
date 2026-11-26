package metrics

import (
	"math"
	"strings"
	"testing"
)

func TestComputeHalsteadCountsOperatorsAndOperands(t *testing.T) {
	h := ComputeHalstead([]byte("func f(a int) int { return a + 1 }"))

	want := Halstead{DistinctOperators: 5, DistinctOperands: 4, TotalOperators: 5, TotalOperands: 6, Vocabulary: 9, Length: 11}
	if h.DistinctOperators != want.DistinctOperators || h.DistinctOperands != want.DistinctOperands ||
		h.TotalOperators != want.TotalOperators || h.TotalOperands != want.TotalOperands ||
		h.Vocabulary != want.Vocabulary || h.Length != want.Length {
		t.Fatalf("counts = %+v, want %+v", h, want)
	}
	assertClose(t, "volume", h.Volume, 11*math.Log2(9))
	assertClose(t, "difficulty", h.Difficulty, 5.0/2*6/4)
	assertClose(t, "effort", h.Effort, h.Difficulty*h.Volume)
	assertClose(t, "bugs", h.Bugs, h.Volume/3000)
}

func TestComputeHalsteadIgnoresCommentsAndLineSemicolons(t *testing.T) {
	plain := ComputeHalstead([]byte("x := 1\ny := x"))
	commented := ComputeHalstead([]byte("x := 1 // first\n/* note */ y := x"))
	if plain != commented {
		t.Fatalf("comments changed metrics: %+v vs %+v", plain, commented)
	}
}

func TestComputeHalsteadEmpty(t *testing.T) {
	if h := ComputeHalstead(nil); h != (Halstead{}) {
		t.Fatalf("empty source = %+v", h)
	}
}

const sample = `package sample

type T struct{}

func (t *T) Branchy(a, b int, ch chan int) int {
	if a > 0 && b > 0 {
		return 1
	}
	for i := 0; i < a; i++ {
		b += i
	}
	switch a {
	case 1:
		return 2
	case 2, 3:
		return 3
	default:
	}
	select {
	case v := <-ch:
		return v
	default:
	}
	return b
}

func Simple() {}

func helper() {
	_ = func() {
		if true {
			return
		}
	}
}
`

func TestAnalyzeSourceCyclomatic(t *testing.T) {
	functions, err := AnalyzeSource("sample.go", []byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, f := range functions {
		got[f.Name] = f.Cyclomatic
	}
	want := map[string]int{"(*T).Branchy": 7, "Simple": 1, "helper": 2}
	for name, cc := range want {
		if got[name] != cc {
			t.Errorf("%s: cyclomatic = %d, want %d", name, got[name], cc)
		}
	}
	if functions[0].Line != 5 || functions[0].Package != "sample" {
		t.Errorf("unexpected position: %+v", functions[0])
	}
}

func TestCheckReportsExceededLimits(t *testing.T) {
	functions := []Function{
		{Name: "ok", Cyclomatic: 10, Halstead: Halstead{Volume: 100, Difficulty: 5}},
		{Name: "bad", Cyclomatic: 11, Halstead: Halstead{Volume: 5000, Difficulty: 50, Effort: 1}},
	}
	violations := Check(functions, Limits{Cyclomatic: 10, Volume: 3000, Difficulty: 30})
	if len(violations) != 3 {
		t.Fatalf("violations = %v", violations)
	}
	for _, v := range violations {
		if v.Function.Name != "bad" {
			t.Errorf("unexpected violation %s", v)
		}
	}
	if !strings.Contains(violations[0].String(), "cyclomatic complexity = 11.0 > 10") {
		t.Errorf("unexpected message %q", violations[0])
	}
}

func TestSummarize(t *testing.T) {
	s := Summarize([]Function{
		{Cyclomatic: 2, Maintainability: 80, Halstead: Halstead{Volume: 10, Bugs: 1}},
		{Cyclomatic: 4, Maintainability: 60, Halstead: Halstead{Volume: 30, Bugs: 2}},
	})
	if s.Functions != 2 || s.MaxCyclomatic != 4 || s.MeanCyclomatic != 3 || s.MaxVolume != 30 ||
		s.TotalVolume != 40 || s.EstimatedBugs != 3 || s.MinMaintainability != 60 {
		t.Fatalf("summary = %+v", s)
	}
}

func assertClose(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}
