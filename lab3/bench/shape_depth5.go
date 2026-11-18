//go:build !baseline

package main

const inputDepth = 5
const inputPattern = "inputs/patient-%d.json"

type Record struct {
	ID       int64
	Name     string
	Email    string
	Active   bool
	Visits   []Visit
	Clinic   Clinic
	Referral *Referral `json:",omitempty"`
}

// Referral - цепочка направлений, задаёт глубину вложенности в профиле formats.
type Referral struct {
	Level  int
	Doctor string
	Next   *Referral `json:",omitempty"`
}

type Clinic struct {
	Name       string
	Department Department
}
type Department struct {
	Name string
	Team Team
}
type Team struct {
	ID     int
	Doctor Doctor
}
type Doctor struct {
	ID   int
	Name string
}
