package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
)

func containers(v any) int {
	count := 1
	switch node := v.(type) {
	case map[string]any:
		for _, child := range node {
			count += containers(child)
		}
	case []any:
		for _, child := range node {
			count += containers(child)
		}
	default:
		return 0
	}
	return count
}

const formatDepth = 100
const formatVisits = 4

func loadFormats() (map[int]Record, []InputInfo, [][]byte, error) {
	records := make(map[int]Record)
	var inputs []InputInfo
	var files [][]byte
	for i, style := range []string{"compact", "pretty"} {
		path := "inputs/" + style + "-100.json"
		raw, err := inputFiles.ReadFile(path)
		if err != nil {
			return nil, nil, nil, err
		}
		record, depth, err := parseRecord(raw, formatVisits, formatDepth)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%s: %w", path, err)
		}
		var tree any
		if err := json.Unmarshal(raw, &tree); err != nil {
			return nil, nil, nil, err
		}
		count := containers(tree)
		multiline := bytes.Contains(bytes.TrimSpace(raw), []byte("\n"))
		if depth != formatDepth || multiline != (style == "pretty") {
			return nil, nil, nil, fmt.Errorf("%s: expected depth %d and %s layout", path, formatDepth, style)
		}
		if i > 0 && !reflect.DeepEqual(records[0], record) {
			return nil, nil, nil, fmt.Errorf("JSON files contain different data")
		}
		records[i] = record
		inputs = append(inputs, InputInfo{File: path, Size: len(record.Visits), Depth: depth, Nodes: count, Format: style, SHA256: fmt.Sprintf("%x", sha256.Sum256(raw))})
		files = append(files, raw)
	}
	return records, inputs, files, nil
}
