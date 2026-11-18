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
type codec struct {
	name        string
	serialize   func(Record) ([]byte, error)
	deserialize func([]byte) (Record, error)
}

func serializeJSON(r Record) ([]byte, error) {
	return json.Marshal(r)
}

func deserializeJSON(data []byte) (Record, error) {
	var r Record
	err := json.Unmarshal(data, &r)
	return r, err
}

func serializeGob(r Record) ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(r)
	return buf.Bytes(), err
}

func deserializeGob(data []byte) (Record, error) {
	var r Record
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(&r)
	return r, err
}

var codecs = []codec{
	{"json", serializeJSON, deserializeJSON},
	{"gob", serializeGob, deserializeGob},
}

var sinkBytes []byte
var sinkRecord Record
var sinkError error

func operation(c codec, scenario string, r Record, encoded []byte) func() {
	switch scenario {
	case "serialize":
		return func() { sinkBytes, sinkError = c.serialize(r) }
	case "deserialize":
		return func() { sinkRecord, sinkError = c.deserialize(encoded) }
	case "roundtrip":
		return func() {
			sinkBytes, sinkError = c.serialize(r)
			if sinkError == nil {
				sinkRecord, sinkError = c.deserialize(sinkBytes)
			}
		}
	default:
		panic("unknown scenario")
	}
}
func validate(c codec, r Record) ([]byte, error) {
	b, err := c.serialize(r)
	if err != nil {
		return nil, err
	}
	got, err := c.deserialize(b)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(got, r) {
		return nil, fmt.Errorf("%s: roundtrip mismatch", c.name)
	}
	return b, nil
}
