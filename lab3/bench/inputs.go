package main

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed inputs/*.json
var inputFiles embed.FS

type InputInfo struct {
	Format string `json:"format,omitempty"`
	Nodes  int    `json:"nodes,omitempty"`
	File   string `json:"file"`
	Size   int    `json:"visits"`
	Depth  int    `json:"depth"`
	SHA256 string `json:"sha256"`
}

func jsonDepth(value any) int {
	maxChild := 0
	switch node := value.(type) {
	case map[string]any:
		for _, child := range node {
			maxChild = max(maxChild, jsonDepth(child))
		}
	case []any:
		for _, child := range node {
			maxChild = max(maxChild, jsonDepth(child))
		}
	default:
		return 0
	}
	return 1 + maxChild
}

func parsePatient(data []byte, size int) (Record, int, error) {
	return parseRecord(data, size, inputDepth)
}

func parseRecord(data []byte, size, wantDepth int) (Record, int, error) {
	var tree any
	if err := json.Unmarshal(data, &tree); err != nil {
		return Record{}, 0, err
	}
	depth := jsonDepth(tree)
	if depth != wantDepth {
		return Record{}, depth, fmt.Errorf("want JSON depth %d, got %d", wantDepth, depth)
	}
	var r Record
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&r); err != nil {
		return Record{}, depth, err
	}
	if len(r.Visits) != size {
		return Record{}, depth, fmt.Errorf("want %d visits, got %d", size, len(r.Visits))
	}
	return r, depth, nil
}

func loadPatients() (map[int]Record, []InputInfo, error) {
	records := make(map[int]Record)
	var inputs []InputInfo
	for _, size := range []int{4, 64} {
		path := fmt.Sprintf(inputPattern, size)
		data, err := inputFiles.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		record, depth, err := parsePatient(data, size)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", path, err)
		}
		records[size] = record
		inputs = append(inputs, InputInfo{File: path, Size: size, Depth: depth,
			SHA256: fmt.Sprintf("%x", sha256.Sum256(data))})
	}
	return records, inputs, nil
}
