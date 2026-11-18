package hashmap_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"maps"
	"testing"
)

func TestFixedMap(t *testing.T) {
	const size = 100_000
	const fingerprint = "2c1a91a495aada7367f113d0cc56594d043c7172956b043ac0bb1dc1b291fd53"
	input := make(map[string]int, size)
	for i := 0; i < size; i++ {
		input[fmt.Sprintf("procedure-%06d", i)] = i
	}

	
	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(encoded))
	if sum != fingerprint {
		t.Fatalf("fixture changed: SHA-256=%s, want %s", sum, fingerprint)
	}
	t.Logf("%d entries, SHA-256=%s", len(input), sum)

	check := func(t *testing.T, decoded map[string]int) {
		t.Helper()
		if !maps.Equal(input, decoded) {
			t.Fatalf("roundtrip changed map contents (got %d entries, want %d)", len(decoded), size)
		}
		t.Run("existing_key", func(t *testing.T) {
			const key = "procedure-099999"
			value, found := decoded[key]
			if !found || value != 99_999 {
				t.Fatalf("%s: got (%d, %t), want (99999, true)", key, value, found)
			}
			t.Logf("%s: value=%d, found=%t", key, value, found)
		})
		t.Run("existing_zero", func(t *testing.T) {
			const key = "procedure-000000"
			value, found := decoded[key]
			if !found || value != 0 {
				t.Fatalf("%s: got (%d, %t), want (0, true)", key, value, found)
			}
			t.Logf("%s: value=%d, found=%t", key, value, found)
		})
		t.Run("missing_key", func(t *testing.T) {
			const key = "procedure-100000"
			value, found := decoded[key]
			if found || value != 0 {
				t.Fatalf("%s: got (%d, %t), want (0, false)", key, value, found)
			}
			t.Logf("%s: value=%d, found=%t", key, value, found)
		})
	}

	t.Run("json", func(t *testing.T) {
		var decoded map[string]int
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatal(err)
		}
		check(t, decoded)
	})
	t.Run("gob", func(t *testing.T) {
		var buffer bytes.Buffer
		if err := gob.NewEncoder(&buffer).Encode(input); err != nil {
			t.Fatal(err)
		}
		var decoded map[string]int
		if err := gob.NewDecoder(&buffer).Decode(&decoded); err != nil {
			t.Fatal(err)
		}
		check(t, decoded)
	})
}
