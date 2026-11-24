package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"lab6/internal/bench"
	"lab6/internal/compare"
	"lab6/internal/grafana"
	"lab6/internal/jaeger"
	"lab6/internal/k6"
	"lab6/internal/variant"
)

const (
	runner      = "bin/runner"
	apiImage    = "vitaline-lab6-api:base"
	clinicDir   = "../project/VitaLineClinic"
	resultsDir  = "results"
	evidenceDir = "evidence"
	grafanaDir  = "results/grafana"
	jaegerURL   = "http://127.0.0.1:16686"
	service     = "vitaline-clinic-api"
	traceLimit  = 1000
	variantFile = "variants.json"
)

var suiteScenarios = []string{"ramp", "stress", "stable"}

const usage = `lab6 — трассировка и логирование VitaLine Clinic

  lab6 run [-scale 1] -variant <вариант> <сценарий>...   прогнать сценарии ЛР4 в одном варианте
  lab6 suite [-scale 1]                                  все сценарии во всех вариантах, затем сравнение
  lab6 bench [-count 10] [-benchtime 3000x]              бенчмарки стоимости трассировки и логирования
  lab6 compare                                           собрать results/index.html
  lab6 after-load [-traces]                              перевести вывод k6 в формат отчёта ЛР4 и выгрузить трейсы (вызывается стендом)
  lab6 traces                                            выгрузить трейсы из Jaeger в $RUN_DIR
  lab6 grafana [-results results]                        подготовить метрики и трейсы для дашборда в results/grafana
  lab6 grafana-traces [-otlp http://127.0.0.1:4319]      загрузить трейсы в Jaeger дашборда
`

func main() {
	if err := dispatch(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "ошибка:", err)
		os.Exit(1)
	}
}

func dispatch(args []string) error {
	if len(args) == 0 {
		fmt.Print(usage)
		return errors.New("не указана команда")
	}
	variants, err := variant.Load(variantFile)
	if err != nil {
		return err
	}
	command, rest := args[0], args[1:]
	flags := flag.NewFlagSet(command, flag.ExitOnError)
	switch command {
	case "run":
		scale := flags.Float64("scale", 1, "множитель длительности фаз")
		name := flags.String("variant", "", "вариант из variants.json")
		flags.Parse(rest)
		v, err := variants.Get(*name)
		if err != nil {
			return err
		}
		return run(v, *scale, flags.Args())
	case "suite":
		scale := flags.Float64("scale", 1, "множитель длительности фаз")
		flags.Parse(rest)
		for _, scenario := range suiteScenarios {
			for _, v := range variants {
				if err := run(v, *scale, []string{scenario}); err != nil {
					return err
				}
			}
		}
		return report(variants)
	case "bench":
		count := flags.Int("count", 10, "число повторов")
		benchtime := flags.String("benchtime", "3000x", "итераций в повторе")
		flags.Parse(rest)
		result, err := bench.Run(clinicDir, resultsDir, *count, *benchtime)
		if err != nil {
			return err
		}
		for _, r := range result.Results {
			fmt.Printf("  %-16s %8.1f мкс/оп  %7.1f КБ/оп  %5.0f аллокаций/оп\n", r.Variant, r.NsPerOp.Mean/1000, r.BytesPerOp.Mean/1024, r.AllocsPerOp.Mean)
		}
		return report(variants)
	case "compare":
		return report(variants)
	case "traces":
		return exportTraces()
	case "after-load":
		traces := flags.Bool("traces", false, "также выгрузить трейсы из Jaeger")
		flags.Parse(rest)
		return afterLoad(*traces)
	case "grafana":
		source := flags.String("results", "", "каталог с прогонами: results, а если в нём пусто — evidence")
		flags.Parse(rest)
		from := *source
		if from == "" {
			from = resultsDir
			if runs, _ := compare.LoadRuns(resultsDir); len(runs) == 0 {
				from = evidenceDir
			}
		}
		result, err := grafana.Export(from, grafanaDir, variants)
		if err != nil {
			return err
		}
		fmt.Printf("Grafana: %d прогонов из %s, %d точек, %d трейсов → %s\n", result.Runs, from, result.Samples, result.Traces, result.Dir)
		return nil
	case "grafana-traces":
		endpoint := flags.String("otlp", "http://127.0.0.1:4319", "OTLP/HTTP Jaeger из grafana/compose.yaml")
		flags.Parse(rest)
		files, err := grafana.PushTraces(grafanaDir, *endpoint)
		if err != nil {
			return fmt.Errorf("загрузка трейсов в Jaeger: %w", err)
		}
		fmt.Printf("Трейсы загружены в Jaeger: %d запросов OTLP\n", files)
		return nil
	default:
		fmt.Print(usage)
		return fmt.Errorf("неизвестная команда %q", command)
	}
}

func run(v variant.Variant, scale float64, scenarios []string) error {
	if len(scenarios) == 0 {
		return errors.New("укажите сценарии")
	}
	args := []string{
		"-lab", ".",
		"-results", filepath.Join(resultsDir, v.Name),
		"-variant", v.Name,
		"-api-image", apiImage,
	}
	for _, pair := range v.Settings() {
		args = append(args, "-env", pair)
	}
	for _, profile := range v.Profiles {
		args = append(args, "-profile", profile)
	}
	self, err := filepath.Abs(os.Args[0])
	if err != nil {
		return err
	}
	hook := strconv.Quote(self) + " after-load"
	if v.Traces {
		hook += " -traces"
	}
	args = append(args, "-after-load", hook)
	args = append(args, "run", "-scale", strconv.FormatFloat(scale, 'g', -1, 64))
	args = append(args, scenarios...)

	fmt.Printf("### вариант %s: %s\n", v.Name, v.Title)
	cmd := exec.Command(runner, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("вариант %s: %w", v.Name, err)
	}
	return nil
}

// afterLoad runs on the host after k6 has finished and before the lab4 report is built.
func afterLoad(traces bool) error {
	runDir := os.Getenv("RUN_DIR")
	if runDir == "" {
		return errors.New("RUN_DIR не задан")
	}
	result, err := k6.Convert(filepath.Join(runDir, "load"))
	if err != nil {
		return fmt.Errorf("вывод k6: %w", err)
	}
	fmt.Printf("k6: %d запросов, неожиданных ответов %d, намеренно ошибочных %d\n", result.Requests, result.Failed, result.Invalid)
	if traces {
		return exportTraces()
	}
	return nil
}

func exportTraces() error {
	runDir := os.Getenv("RUN_DIR")
	if runDir == "" {
		return errors.New("RUN_DIR не задан")
	}
	summary, err := jaeger.NewClient(jaegerURL).Export(service, traceLimit, runDir)
	if err != nil {
		return fmt.Errorf("выгрузка трейсов: %w", err)
	}
	fmt.Printf("Трейсы: %d трейсов, %d спанов → %s\n", summary.Traces, summary.Spans, filepath.Join(runDir, jaeger.TracesDir))
	return nil
}

func report(variants variant.Set) error {
	page, err := compare.Build(resultsDir, variants)
	if err != nil {
		return err
	}
	fmt.Println("Сравнение:", page)
	return nil
}
