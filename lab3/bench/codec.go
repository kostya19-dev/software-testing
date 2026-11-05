package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"reflect"
)

type Visit struct {
	Code     string
	Duration int
	Price    int64
}
type Record struct {
	ID     int64
	Name   string
	Email  string
	Active bool
	Visits []Visit
}

func fixture(size int) Record {
	r := Record{ID: 42, Name: "Учебный пациент", Email: "patient@example.invalid", Active: true}
	for i := 0; i < size; i++ {
		r.Visits = append(r.Visits, Visit{fmt.Sprintf("procedure-%03d", i), 15 + i%5, int64(1200 + i*17)})
	}
	return r
}

type codec struct {
	name   string
	encode func(Record) ([]byte, error)
	decode func([]byte) (Record, error)
}

var codecs = []codec{
	{"json", func(r Record) ([]byte, error) { return json.Marshal(r) }, func(b []byte) (r Record, err error) { err = json.Unmarshal(b, &r); return }},
	{"gob", func(r Record) ([]byte, error) {
		var b bytes.Buffer
		err := gob.NewEncoder(&b).Encode(r)
		return b.Bytes(), err
	},
		func(b []byte) (r Record, err error) { err = gob.NewDecoder(bytes.NewReader(b)).Decode(&r); return }},
}

// Observable sinks prevent dead-code elimination. No I/O in timed code.
var sinkBytes []byte
var sinkRecord Record
var sinkError error

func operation(c codec, scenario string, r Record, encoded []byte) func() {
	switch scenario {
	case "write":
		return func() { sinkBytes, sinkError = c.encode(r) }
	case "read":
		return func() { sinkRecord, sinkError = c.decode(encoded) }
	case "roundtrip":
		return func() {
			sinkBytes, sinkError = c.encode(r)
			if sinkError == nil {
				sinkRecord, sinkError = c.decode(sinkBytes)
			}
		}
	default:
		panic("unknown scenario")
	}
}
func validate(c codec, r Record) ([]byte, error) {
	b, err := c.encode(r)
	if err != nil {
		return nil, err
	}
	got, err := c.decode(b)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(got, r) {
		return nil, fmt.Errorf("%s: roundtrip mismatch", c.name)
	}
	return b, nil
}
