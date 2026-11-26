// Package metrics computes per-function cyclomatic complexity (McCabe) and
// Halstead complexity metrics for Go source code.
package metrics

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Halstead holds the base counts and derived Halstead metrics of a code unit.
//
// Operands are identifiers and literals. Operators are every other token:
// keywords, arithmetic/logical operators and punctuation. Paired delimiters
// "()", "[]", "{}" count once (by the opening token). Semicolons inserted by
// the Go scanner at line ends are not counted.
type Halstead struct {
	DistinctOperators int     `json:"n1"`
	DistinctOperands  int     `json:"n2"`
	TotalOperators    int     `json:"N1"`
	TotalOperands     int     `json:"N2"`
	Vocabulary        int     `json:"vocabulary"`
	Length            int     `json:"length"`
	Volume            float64 `json:"volume"`
	Difficulty        float64 `json:"difficulty"`
	Effort            float64 `json:"effort"`
	TimeSeconds       float64 `json:"time_seconds"`
	Bugs              float64 `json:"bugs"`
}

// Function describes the metrics of one function or method.
type Function struct {
	Package    string   `json:"package"`
	Name       string   `json:"name"`
	File       string   `json:"file"`
	Line       int      `json:"line"`
	Lines      int      `json:"lines"`
	Cyclomatic int      `json:"cyclomatic"`
	Halstead   Halstead `json:"halstead"`
	// Maintainability is the Microsoft-normalised maintainability index (0..100).
	Maintainability float64 `json:"maintainability"`
}

// Location returns "file:line" for messages.
func (f Function) Location() string { return fmt.Sprintf("%s:%d", f.File, f.Line) }

// Options control which files are analysed.
type Options struct {
	IncludeTests bool
	Exclude      []string // path fragments to skip, e.g. "vendor/"
}

// AnalyzeDir walks root and analyses every Go file in it.
func AnalyzeDir(root string, opts Options) ([]Function, error) {
	var result []Function
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip(path, d, opts) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		src, err := os.ReadFile(path) //nolint:gosec // G304: walking the analysed source tree
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		functions, err := AnalyzeSource(filepath.ToSlash(rel), src)
		if err != nil {
			return err
		}
		result = append(result, functions...)
		return nil
	})
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].File != result[j].File {
			return result[i].File < result[j].File
		}
		return result[i].Line < result[j].Line
	})
	return result, err
}

func skip(path string, d os.DirEntry, opts Options) bool {
	if d.IsDir() {
		if path != "." && hiddenOrVendored(d.Name()) {
			return true
		}
	} else if !opts.IncludeTests && strings.HasSuffix(d.Name(), "_test.go") {
		return true
	}
	return excluded(filepath.ToSlash(path), opts.Exclude)
}

func hiddenOrVendored(dir string) bool {
	return strings.HasPrefix(dir, ".") || dir == "vendor" || dir == "testdata"
}

func excluded(path string, fragments []string) bool {
	for _, fragment := range fragments {
		if fragment != "" && strings.Contains(path, fragment) {
			return true
		}
	}
	return false
}

// AnalyzeSource analyses a single Go file.
func AnalyzeSource(filename string, src []byte) ([]Function, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var result []Function
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		start, end := fset.Position(fn.Pos()), fset.Position(fn.End())
		h := ComputeHalstead(src[start.Offset:end.Offset])
		cc := Cyclomatic(fn)
		lines := end.Line - start.Line + 1
		result = append(result, Function{
			Package:         file.Name.Name,
			Name:            funcName(fn),
			File:            filename,
			Line:            start.Line,
			Lines:           lines,
			Cyclomatic:      cc,
			Halstead:        h,
			Maintainability: maintainability(h.Volume, cc, lines),
		})
	}
	return result, nil
}

func funcName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	return "(" + typeName(fn.Recv.List[0].Type) + ")." + fn.Name.Name
}

func typeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return "*" + typeName(t.X)
	case *ast.IndexExpr:
		return typeName(t.X)
	case *ast.IndexListExpr:
		return typeName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return "?"
}

// Cyclomatic returns McCabe's cyclomatic complexity using the same rules as
// gocyclo: 1 + if, for, range, non-default case/comm clauses, && and ||.
func Cyclomatic(fn *ast.FuncDecl) int {
	complexity := 1
	ast.Inspect(fn, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
			complexity++
		case *ast.CaseClause:
			if node.List != nil {
				complexity++
			}
		case *ast.CommClause:
			if node.Comm != nil {
				complexity++
			}
		case *ast.BinaryExpr:
			if node.Op == token.LAND || node.Op == token.LOR {
				complexity++
			}
		}
		return true
	})
	return complexity
}

// closing delimiters are counted together with their opening pair.
var closing = map[token.Token]bool{token.RPAREN: true, token.RBRACK: true, token.RBRACE: true}

// ComputeHalstead tokenises src and computes Halstead metrics.
func ComputeHalstead(src []byte) Halstead {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var s scanner.Scanner
	s.Init(file, src, nil, 0)

	operators, operands := map[string]int{}, map[string]int{}
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		switch {
		case tok == token.SEMICOLON && lit == "\n", closing[tok], tok == token.COMMENT:
			continue
		case tok == token.IDENT || tok.IsLiteral():
			operands[lit]++
		default:
			operators[tok.String()]++
		}
	}
	return newHalstead(operators, operands)
}

func newHalstead(operators, operands map[string]int) Halstead {
	h := Halstead{DistinctOperators: len(operators), DistinctOperands: len(operands)}
	for _, count := range operators {
		h.TotalOperators += count
	}
	for _, count := range operands {
		h.TotalOperands += count
	}
	h.Vocabulary = h.DistinctOperators + h.DistinctOperands
	h.Length = h.TotalOperators + h.TotalOperands
	if h.Vocabulary > 0 {
		h.Volume = float64(h.Length) * math.Log2(float64(h.Vocabulary))
	}
	if h.DistinctOperands > 0 {
		h.Difficulty = float64(h.DistinctOperators) / 2 * float64(h.TotalOperands) / float64(h.DistinctOperands)
	}
	h.Effort = h.Difficulty * h.Volume
	h.TimeSeconds = h.Effort / 18
	h.Bugs = h.Volume / 3000
	return h
}

// maintainability is the Microsoft variant of the maintainability index:
// max(0, (171 - 5.2 ln V - 0.23 CC - 16.2 ln LOC) * 100 / 171).
func maintainability(volume float64, cyclomatic, lines int) float64 {
	if volume <= 0 || lines <= 0 {
		return 100
	}
	mi := (171 - 5.2*math.Log(volume) - 0.23*float64(cyclomatic) - 16.2*math.Log(float64(lines))) * 100 / 171
	return math.Max(0, mi)
}
