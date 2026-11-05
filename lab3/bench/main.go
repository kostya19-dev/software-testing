package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"
)

type Trial struct {
	Schema     int      `json:"schema"`
	Trial      int      `json:"trial"`
	Seed       int64    `json:"seed"`
	Go         string   `json:"go"`
	OS         string   `json:"os"`
	Arch       string   `json:"arch"`
	GOMAXPROCS int      `json:"gomaxprocs"`
	Started    string   `json:"started"`
	Overhead   Overhead `json:"overhead"`
	Results    []Result `json:"results"`
	ElapsedNS  int64    `json:"elapsed_ns"`
}

func main() {
	trial := flag.Int("trial", 1, "positive trial number")
	n := flag.Int("iterations", 256, "operations per warmup, batch and teardown")
	samples := flag.Int("samples", 256, "individually timed requests per case")
	seed := flag.Int64("seed", 20261105, "reproducible order seed")
	flag.Parse()
	if *trial < 1 || *n < 1 || *samples < 1 {
		fmt.Fprintln(os.Stderr, "positive trial, iterations and samples required")
		os.Exit(2)
	}
	runtime.GOMAXPROCS(1)
	epoch := time.Now()
	out := Trial{Schema: 1, Trial: *trial, Seed: *seed, Go: runtime.Version(), OS: runtime.GOOS, Arch: runtime.GOARCH, GOMAXPROCS: runtime.GOMAXPROCS(0), Started: epoch.UTC().Format(time.RFC3339Nano)}
	out.Overhead = overhead()
	type scenario struct {
		name string
		size int
	}
	cases := []scenario{}
	for _, size := range []int{4, 64} {
		for _, name := range []string{"write", "read", "roundtrip"} {
			cases = append(cases, scenario{name, size})
		}
	}
	rng := rand.New(rand.NewSource(*seed + int64(*trial)))
	rng.Shuffle(len(cases), func(i, j int) { cases[i], cases[j] = cases[j], cases[i] })
	for _, sc := range cases {
		r := fixture(sc.size)

		order := []int{0, 1}
		if *trial%2 == 0 {
			order = []int{1, 0}
		}
		for _, idx := range order {
			c := codecs[idx]
			encoded, err := validate(c, r)
			if err != nil {
				panic(err)
			}
			got := measure(operation(c, sc.name, r, encoded), *n, *samples, epoch)
			if sinkError != nil {
				panic(sinkError)
			}
			got.Codec, got.Scenario, got.Size, got.Bytes, got.Order = c.name, sc.name, sc.size, len(encoded), len(out.Results)
			out.Results = append(out.Results, got)
		}
	}
	out.ElapsedNS = time.Since(epoch).Nanoseconds()
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		panic(err)
	}
}
