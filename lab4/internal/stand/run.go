package stand

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const manifestFile = "manifest.json"

type Service struct {
	Image      string  `json:"image"`
	ImageID    string  `json:"image_id"`
	Cpuset     string  `json:"cpuset"`
	CPUs       float64 `json:"cpus"`
	MemoryMB   int64   `json:"memory_mb"`
	ReservedMB int64   `json:"memory_reserved_mb,omitempty"`
}

type Run struct {
	ID         string             `json:"id"`
	Scenario   string             `json:"scenario"`
	Scale      float64            `json:"scale"`
	StartedAt  string             `json:"started_at"`
	FinishedAt string             `json:"finished_at"`
	ExitCode   int                `json:"load_exit_code"`
	Variant    string             `json:"variant,omitempty"`
	Settings   map[string]string  `json:"settings,omitempty"`
	Services   map[string]Service `json:"services"`
	Dir        string             `json:"-"`
}

func CreateRun(resultsDir, scenario string, scale float64) (*Run, error) {
	id := fmt.Sprintf("%s-%s-%s", time.Now().Format("20060102-150405"), scenario, strings.ToLower(rand.Text()[:4]))
	r := &Run{ID: id, Scenario: scenario, Scale: scale, Services: map[string]Service{}, Dir: filepath.Join(resultsDir, id)}
	if err := os.MkdirAll(r.LoadDir(), 0o755); err != nil {
		return nil, err
	}
	return r, r.Save()
}

func OpenRun(dir string) (*Run, error) {
	data, err := os.ReadFile(filepath.Join(dir, manifestFile))
	if err != nil {
		return nil, err
	}
	var r Run
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	r.Dir = dir
	return &r, nil
}

func (r *Run) LoadDir() string {
	return filepath.Join(r.Dir, "load")
}

func (r *Run) Path(name string) string {
	return filepath.Join(r.Dir, name)
}

func (r *Run) MarkStarted() error {
	r.StartedAt = now()
	return r.Save()
}

func (r *Run) MarkFinished(exitCode int) error {
	r.FinishedAt = now()
	r.ExitCode = exitCode
	return r.Save()
}

func (r *Run) Save() error {
	return writeJSON(r.Path(manifestFile), r)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func now() string {
	return time.Now().Format(time.RFC3339)
}
