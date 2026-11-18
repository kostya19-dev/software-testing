//go:build baseline

package main

const inputDepth = 3
const inputPattern = "inputs/baseline-%d.json"

type Record struct {
	ID     int64
	Name   string
	Email  string
	Active bool
	Visits []Visit
}
