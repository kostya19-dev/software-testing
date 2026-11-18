package stand

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	apiImage       = "vitaline-lab4-api"
	APIBaseImage   = apiImage + ":base"
	LoadgenImage   = "vitaline-lab4-loadgen:1"
	ExporterImage  = "vitaline-lab4-exporter:1"
	prometheusPort = 9094
	scrapeSettle   = 5 * time.Second
)

type Paths struct {
	Lab    string
	Clinic string
}

type Options struct {
	APIImage string
	Profiles []string
	Env      []string
}

type Stand struct {
	run        *Run
	compose    compose
	apiImage   string
	Prometheus *Client
}

func Up(paths Paths, r *Run, opts Options) (*Stand, error) {
	base := cmp.Or(opts.APIImage, APIBaseImage)
	repository, _, _ := strings.Cut(base, ":")
	s := &Stand{
		run:        r,
		apiImage:   repository + ":" + r.ID,
		Prometheus: NewPrometheus(fmt.Sprintf("http://127.0.0.1:%d", prometheusPort)),
	}
	s.compose = compose{
		project:  "lab4-" + r.ID,
		file:     filepath.Join(paths.Lab, "stand", "compose.yaml"),
		profiles: opts.Profiles,
		env: append([]string{
			"RUN_ID=" + r.ID,
			"RUN_DIR=" + r.Dir,
			"SCENARIO=" + r.Scenario,
			"TIME_SCALE=" + strconv.FormatFloat(r.Scale, 'g', -1, 64),
			"API_IMAGE=" + s.apiImage,
			"LOADGEN_IMAGE=" + LoadgenImage,
			"EXPORTER_IMAGE=" + ExporterImage,
			"CLINIC_ROOT=" + paths.Clinic,
			"PROMETHEUS_PORT=" + strconv.Itoa(prometheusPort),
			"HOST_UID=" + strconv.Itoa(os.Getuid()),
			"HOST_GID=" + strconv.Itoa(os.Getgid()),
		}, opts.Env...),
	}

	if err := ensureFree(); err != nil {
		return nil, err
	}
	if _, err := docker("image", "tag", base, s.apiImage); err != nil {
		return nil, fmt.Errorf("нет образа %s, выполните make setup: %w", base, err)
	}
	if err := s.recordConfiguration(); err != nil {
		return nil, errors.Join(err, s.Down())
	}
	if err := s.compose.run("up", "--detach", "--wait", "--quiet-pull"); err != nil {
		return nil, errors.Join(err, s.Down())
	}
	if err := s.Prometheus.WaitReady(time.Minute); err != nil {
		return nil, errors.Join(err, s.Down())
	}
	return s, nil
}

func (s *Stand) Load() (int, error) {
	if err := s.run.MarkStarted(); err != nil {
		return 0, err
	}
	err := s.compose.run("run", "--rm", "--no-deps", "--use-aliases", "--name", "lab4-"+s.run.ID+"-loadgen", "loadgen")
	exitCode := 0
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		exitCode = exit.ExitCode()
	case err != nil:
		return 0, err
	}
	time.Sleep(scrapeSettle)
	return exitCode, s.run.MarkFinished(exitCode)
}

func (s *Stand) Down() error {
	logs, _ := s.compose.output("logs", "--no-color", "--tail", "200", "api")
	return errors.Join(
		os.WriteFile(s.run.Path("api.log"), []byte(logs), 0o644),
		s.compose.quiet("down", "--volumes", "--remove-orphans"),
		removeImage(s.apiImage),
	)
}

type composeService struct {
	Image      string   `json:"image"`
	Cpuset     string   `json:"cpuset"`
	CPUs       float64  `json:"cpus"`
	MemLimit   flexible `json:"mem_limit"`
	MemReserve flexible `json:"mem_reservation"`
}

func (s *Stand) recordConfiguration() error {
	out, err := s.compose.output("--profile", "load", "config", "--format", "json")
	if err != nil {
		return err
	}
	var config struct {
		Services map[string]composeService `json:"services"`
	}
	if err := json.Unmarshal([]byte(out), &config); err != nil {
		return err
	}
	for name, service := range config.Services {
		id, err := docker("image", "inspect", "--format", "{{.Id}}", service.Image)
		if err != nil {
			return err
		}
		s.run.Services[name] = Service{
			Image:      service.Image,
			ImageID:    id,
			Cpuset:     service.Cpuset,
			CPUs:       service.CPUs,
			MemoryMB:   int64(service.MemLimit) >> 20,
			ReservedMB: int64(service.MemReserve) >> 20,
		}
	}
	return s.run.Save()
}

func ensureFree() error {
	running, err := docker("ps", "--filter", "label=lab4.run", "--format", `{{.Label "com.docker.compose.project"}}`)
	if err != nil {
		return err
	}
	if running == "" {
		return nil
	}
	project, _, _ := strings.Cut(running, "\n")
	return fmt.Errorf("уже работает стенд %s: закройте make stand клавишей Enter или выполните make clean", project)
}

func removeImage(reference string) error {
	_, err := docker("image", "rm", reference)
	return err
}

type flexible int64

func (f *flexible) UnmarshalJSON(data []byte) error {
	value, err := strconv.ParseInt(strings.Trim(string(data), `"`), 10, 64)
	*f = flexible(value)
	return err
}
