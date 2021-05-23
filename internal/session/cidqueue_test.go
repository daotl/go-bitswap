package session

import (
	"testing"

	"github.com/daotl/go-bitswap/internal/testutil"
)

func Test_newKeyQueue(t *testing.T) {
	kq := newKeyQueue()
	if kq.Len() != 0 {
		t.Error("len should be 0")
	}
	ks := testutil.GenerateWantKeys(3)
	kq.Push(ks[0])
	kq.Push(ks[1])
	expected := 2
	if kq.Len() != expected {
		t.Errorf("len should be %v", expected)
	}
	// test FIFO
	if !kq.Pop().Equals(ks[0]) {
		t.Error("ks[0] should be popped")
	}

	// deduplication
	kq.Push(ks[2])
	kq.Push(ks[2])
	if kq.Len() != expected {
		t.Errorf("len should be %v", expected)
	}

	// has
	if !kq.Has(ks[1]) {
		t.Errorf("ks[1] shoule exist")
	}
	if kq.Has(ks[0]) {
		t.Errorf("ks[0] must not exist")
	}

	// remove
	kq.Remove(ks[1])
	if !kq.Pop().Equals(ks[2]) {
		t.Error("ks[2] should be popped")
	}
}
