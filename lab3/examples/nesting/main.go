package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

const maxDepth = 1000

type Visit struct {
	ID       int    `json:"id"`
	FollowUp *Visit `json:"followUp"`
}

func visitChain(depth int) *Visit {
	var next *Visit
	for id := depth; id > 0; id-- {
		next = &Visit{ID: id, FollowUp: next}
	}
	return next
}

func chainDepth(root *Visit) int {
	depth := 0
	for visit := root; visit != nil; visit = visit.FollowUp {
		depth++
	}
	return depth
}

type codec struct {
	name      string
	roundTrip func(*Visit) ([]byte, *Visit, error)
}

func roundTripJSON(input *Visit) ([]byte, *Visit, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, nil, err
	}
	var output Visit
	err = json.Unmarshal(data, &output)
	return data, &output, err
}

func roundTripGob(input *Visit) ([]byte, *Visit, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(input); err != nil {
		return nil, nil, err
	}
	data := buf.Bytes()
	var output Visit
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(&output)
	return data, &output, err
}

var codecs = []codec{{"JSON", roundTripJSON}, {"gob", roundTripGob}}

func compareVisits(want, got *Visit) error {
	for level := 1; want != nil || got != nil; level++ {
		if want == nil || got == nil {
			return fmt.Errorf("level %d: chain lengths differ", level)
		}
		if got.ID != want.ID {
			return fmt.Errorf("level %d: got id %d, want %d", level, got.ID, want.ID)
		}
		want, got = want.FollowUp, got.FollowUp
	}
	return nil
}

func readVisits(path string) (*Visit, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var visits *Visit
	if err := decoder.Decode(&visits); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return nil, fmt.Errorf("expected exactly one JSON value in %s", path)
	}
	return visits, nil
}

func run() error {
	depth := flag.Int("depth", 3, "number of visit nodes (1..1000)")
	path := flag.String("in", "", "read a visit chain from a JSON file instead of generating it")
	jsonOnly := flag.Bool("json", false, "print only the input JSON")
	flag.Parse()
	if *depth < 1 || *depth > maxDepth {
		return fmt.Errorf("depth must be between 1 and %d", maxDepth)
	}
	var root *Visit
	if *path == "" {
		root = visitChain(*depth)
	} else {
		var err error
		root, err = readVisits(*path)
		if err != nil {
			return err
		}
	}
	n := chainDepth(root)
	if n < 1 || n > maxDepth {
		return fmt.Errorf("expected 1..%d visits, got %d", maxDepth, n)
	}
	if *jsonOnly || n <= 6 {
		encoder := json.NewEncoder(os.Stdout)
		if n <= 6 {
			encoder.SetIndent("", "  ")
		}
		if err := encoder.Encode(root); err != nil || *jsonOnly {
			return err
		}
	}
	fmt.Printf("Input: %d visit levels\n", n)
	for _, c := range codecs {
		data, output, err := c.roundTrip(root)
		if err == nil {
			err = compareVisits(root, output)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", c.name, err)
		}
		fmt.Printf("%s: %d bytes, %d levels, every ID and final nil verified\n", c.name, len(data), n)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
