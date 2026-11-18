package main

import (
	"fmt"
	"testing"
)

func TestNestedRoundTrip(t *testing.T) {
	for _, depth := range []int{1, 10, 100, 1000} {
		input := visitChain(depth)
		if got := chainDepth(input); got != depth {
			t.Fatalf("fixture has %d levels, want %d", got, depth)
		}
		for _, c := range codecs {
			t.Run(fmt.Sprintf("depth_%d/%s", depth, c.name), func(t *testing.T) {
				data, decoded, err := c.roundTrip(input)
				if err != nil {
					t.Fatal(err)
				}
				if err := compareVisits(input, decoded); err != nil {
					t.Fatal(err)
				}
				t.Logf("%d levels, %d encoded bytes; all visit IDs and the final nil match", depth, len(data))
			})
		}
	}
}

func TestDeepTailMismatch(t *testing.T) {
	want, got := visitChain(1000), visitChain(1000)
	tail := got
	for tail.FollowUp != nil {
		tail = tail.FollowUp
	}
	tail.ID++
	if err := compareVisits(want, got); err == nil {
		t.Fatal("change to the last visit went undetected")
	}
	if err := compareVisits(want, visitChain(999)); err == nil {
		t.Fatal("truncation went undetected")
	}
}
