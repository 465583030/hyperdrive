package hashtrie

import (
	"fmt"
	"testing"
)

type test struct {
	bitmap   uint
	expected uint
	index    uint
}

func TestCTPopCountsBits(t *testing.T) {
	data := []test{
		test{0, 0, 0},
		test{1, 1, 0},
		test{3, 2, 0},
		test{25, 3, 0},
	}

	for _, test := range data {
		r := CTPop(test.bitmap)
		if r != test.expected {
			t.Fatalf("expected %d received %d", test.expected, r)
		}
	}
}

func TestCTPopBelowCountsBits(t *testing.T) {
	data := []test{
		test{0, 0, 0},  // 0000, 0 -> 0
		test{1, 1, 1},  // 0001, 1 -> 1
		test{1, 0, 0},  // 0001, 0 -> 0
		test{3, 1, 1},  // 0011, 1 -> 1
		test{15, 3, 3}, // 1111, 3 -> 3
		test{15, 2, 2}, // 1111, 2 -> 2
	}

	for _, test := range data {
		r, err := CTPopBelow(test.bitmap, test.index)

		if err != nil {
			t.Fatalf("expected %d received %v", test.expected, err)
		}

		if r != test.expected {
			t.Fatalf("expected %d received %d", test.expected, r)
		}
	}
}

func TestCTPopBelowCheckBounds(t *testing.T) {
	r, err := CTPopBelow(1, 32)
	if err == nil {
		t.Fatalf("expected an error received %d", r)
	}

	const msg = "Index out of bounds"
	rmsg := fmt.Sprintf("%s", err)

	if rmsg != "Index out of bounds" {
		t.Fatalf("expected an error '%s' received '%s'", msg, rmsg)
	}
}

func BenchmarkCTPopBelow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CTPopBelow(5, 2)
	}
}
