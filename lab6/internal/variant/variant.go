package variant

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
)

type Variant struct {
	Name     string            `json:"name"`
	Title    string            `json:"title"`
	Env      map[string]string `json:"env"`
	Profiles []string          `json:"profiles,omitempty"`
	Traces   bool              `json:"traces,omitempty"`
}

type Set []Variant

func Load(path string) (Set, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var set Set
	if err := json.Unmarshal(data, &set); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return set, nil
}

func (s Set) Get(name string) (Variant, error) {
	for _, v := range s {
		if v.Name == name {
			return v, nil
		}
	}
	return Variant{}, fmt.Errorf("неизвестный вариант %q, есть: %v", name, s.Names())
}

func (s Set) Names() []string {
	names := make([]string, len(s))
	for i, v := range s {
		names[i] = v.Name
	}
	return names
}

func (v Variant) Settings() []string {
	pairs := make([]string, 0, len(v.Env))
	for _, key := range slices.Sorted(maps.Keys(v.Env)) {
		pairs = append(pairs, key+"="+v.Env[key])
	}
	return pairs
}
