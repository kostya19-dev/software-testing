package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"syscall"
	"time"

	"lab4/internal/load"
	"lab4/internal/report"
	"lab4/internal/stand"
)

const resourceMargin = 5.0

const usage = `runner — нагрузочное тестирование VitaLine Clinic

  runner [флаги стенда] run [-scale 1] <сценарий>...   выполнить сценарии, каждый на свежем стенде
  runner [флаги стенда] suite [-scale 1]               выполнить все сценарии по очереди
  runner [флаги стенда] stand                          поднять стенд без нагрузки, Enter — удалить
  runner [флаги стенда] report <каталог прогона>...    пересобрать отчёты по сохранённым данным
  runner [флаги стенда] index                          обновить список прогонов

Флаги стенда:
`

type lab struct {
	paths     stand.Paths
	results   string
	catalog   *load.Catalog
	options   stand.Options
	variant   string
	afterLoad string
}

func main() {
	if err := dispatch(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "ошибка:", err)
		os.Exit(1)
	}
}

func dispatch(args []string) error {
	settings, args, err := parseSettings(args)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.New("не указана команда")
	}
	l, err := newLab(settings)
	if err != nil {
		return err
	}
	command, rest := args[0], args[1:]
	switch command {
	case "run", "suite":
		flags := flag.NewFlagSet(command, flag.ExitOnError)
		scale := flags.Float64("scale", 1, "множитель длительности фаз")
		if err := flags.Parse(rest); err != nil {
			return err
		}
		names := flags.Args()
		if command == "suite" {
			names = l.catalog.Names()
		}
		return l.runAll(names, *scale)
	case "stand":
		return l.hold()
	case "report":
		return l.rebuild(rest)
	case "index":
		return l.index()
	default:
		fmt.Print(usage)
		return fmt.Errorf("неизвестная команда %q", command)
	}
}

func newLab(s settings) (*lab, error) {
	root, err := filepath.Abs(s.dir)
	if err != nil {
		return nil, err
	}
	catalog, err := load.LoadCatalog(filepath.Join(root, "scenarios.json"))
	if err != nil {
		return nil, err
	}
	results := filepath.Join(root, "results")
	if s.results != "" {
		if results, err = filepath.Abs(s.results); err != nil {
			return nil, err
		}
	}
	return &lab{
		paths:     stand.Paths{Lab: root, Clinic: filepath.Join(root, "..", "project", "VitaLineClinic")},
		results:   results,
		catalog:   catalog,
		options:   stand.Options{APIImage: s.apiImage, Profiles: s.profiles, Env: s.env},
		variant:   s.variant,
		afterLoad: s.afterLoad,
	}, nil
}

func (l *lab) runAll(names []string, scale float64) error {
	if len(names) == 0 {
		return fmt.Errorf("укажите сценарии: %v", l.catalog.Names())
	}
	for _, name := range names {
		if _, err := l.catalog.Get(name); err != nil {
			return err
		}
	}
	for _, name := range names {
		if err := l.execute(name, scale); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return l.index()
}

func (l *lab) execute(name string, scale float64) error {
	r, err := stand.CreateRun(l.results, name, scale)
	if err != nil {
		return err
	}
	r.Variant, r.Settings = l.variant, settingsMap(l.options.Env)
	if err := r.Save(); err != nil {
		return err
	}
	fmt.Printf("==> %s: %s\n", l.catalog.Scenarios[name].Title, r.ID)

	if err := l.load(r); err != nil {
		if r.StartedAt == "" {
			err = errors.Join(err, os.RemoveAll(r.Dir))
		}
		return err
	}
	page, err := report.Build(l.catalog, r)
	if err != nil {
		return err
	}
	fmt.Println("Отчёт:", page)
	return nil
}

func (l *lab) load(r *stand.Run) (err error) {
	s, err := stand.Up(l.paths, r, l.options)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, s.Down()) }()
	printLimits(r)

	started := float64(time.Now().UnixMilli()) / 1000
	exitCode, err := s.Load()
	if err != nil {
		return err
	}
	if exitCode != 0 {
		fmt.Printf("генератор нагрузки завершился с кодом %d\n", exitCode)
	}
	finished := float64(time.Now().UnixMilli()) / 1000
	usage, err := stand.CollectResources(s.Prometheus, started-resourceMargin, finished)
	if err != nil {
		return err
	}
	if err := usage.Save(report.ResourcesPath(r)); err != nil {
		return err
	}
	return l.runAfterLoad(r)
}

func (l *lab) runAfterLoad(r *stand.Run) error {
	if l.afterLoad == "" {
		return nil
	}
	cmd := exec.Command("sh", "-c", l.afterLoad)
	cmd.Env = append(os.Environ(), "RUN_DIR="+r.Dir)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("after-load: %w", err)
	}
	return nil
}

func (l *lab) hold() (err error) {
	dir, err := os.MkdirTemp("", "lab4-stand-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	r, err := stand.CreateRun(dir, "stand", 1)
	if err != nil {
		return err
	}
	s, err := stand.Up(l.paths, r, l.options)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, s.Down()) }()

	printLimits(r)
	fmt.Printf("Стенд lab4-%s поднят. Prometheus: http://localhost:9094\n", r.ID)
	fmt.Println("Enter — остановить и удалить стенд")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		bufio.NewReader(os.Stdin).ReadString('\n')
		stop()
	}()
	<-ctx.Done()
	return nil
}

func printLimits(r *stand.Run) {
	names := slices.Sorted(maps.Keys(r.Services))
	fmt.Println("Ресурсы стенда:")
	for _, name := range names {
		s := r.Services[name]
		quota := "без квоты"
		if s.CPUs > 0 {
			quota = fmt.Sprintf("квота CPU %g", s.CPUs)
		}
		fmt.Printf("  %-11s CPU %-4s %-14s RAM резерв %d МБ, лимит %d МБ\n", name, s.Cpuset, quota, s.ReservedMB, s.MemoryMB)
	}
}

func (l *lab) rebuild(dirs []string) error {
	for _, dir := range dirs {
		r, err := stand.OpenRun(dir)
		if err != nil {
			return err
		}
		page, err := report.Build(l.catalog, r)
		if err != nil {
			return err
		}
		fmt.Println("Отчёт:", page)
	}
	return l.index()
}

func (l *lab) index() error {
	page, err := report.BuildIndex(l.results)
	if err == nil {
		fmt.Println("Список прогонов:", page)
	}
	return err
}
