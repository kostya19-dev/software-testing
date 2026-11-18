package stand

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func docker(args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("docker", args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

type compose struct {
	project  string
	file     string
	env      []string
	profiles []string
}

func (c compose) command(args ...string) *exec.Cmd {
	base := []string{"compose", "-p", c.project, "-f", c.file}
	for _, profile := range c.profiles {
		base = append(base, "--profile", profile)
	}
	cmd := exec.Command("docker", append(base, args...)...)
	cmd.Env = append(os.Environ(), c.env...)
	cmd.Env = append(cmd.Env, "COMPOSE_PROJECT_NAME="+c.project)
	return cmd
}

func (c compose) run(args ...string) error {
	cmd := c.command(args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func (c compose) output(args ...string) (string, error) {
	var stderr bytes.Buffer
	cmd := c.command(args...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker compose %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

func (c compose) quiet(args ...string) error {
	_, err := c.output(args...)
	return err
}
