package wantlist

import (
	"testing"

	"github.com/daotl/go-ipld-channel/pair"
	"github.com/ipfs/go-cid"

	pb "github.com/daotl/go-bitswap/message/pb"
)

var testpairs []pair.CidChannelPair

func init() {
	strs := []string{
		"QmQL8LqkEgYXaDHdNYCG2mmpow7Sp8Z8Kt3QS688vyBeC7",
		"QmcBDsdjgSXU7BP4A4V8LJCXENE5xVwnhrhRGVTJr9YCVj",
		"QmQakgd2wDxc3uUF4orGdEm28zUT9Mmimp5pyPG2SFS9Gj",
	}
	for _, s := range strs {
		c, err := cid.Decode(s)
		if err != nil {
			panic(err)
		}
		testpairs = append(testpairs, pair.PublicCidPair(c))
	}

}

type wli interface {
	Contains(pair.CidChannelPair) (Entry, bool)
}

func assertHasPair(t *testing.T, w wli, p pair.CidChannelPair) {
	e, ok := w.Contains(p)
	if !ok {
		t.Fatal("expected to have ", p)
	}
	if !e.Pair.Equals(p) {
		t.Fatal("returned entry had wrong CidChannelPair value")
	}
}

func TestBasicWantlist(t *testing.T) {
	wl := New()

	if !wl.Add(testpairs[0], 5, pb.Message_Wantlist_Block) {
		t.Fatal("expected true")
	}
	assertHasPair(t, wl, testpairs[0])
	if !wl.Add(testpairs[1], 4, pb.Message_Wantlist_Block) {
		t.Fatal("expected true")
	}
	assertHasPair(t, wl, testpairs[0])
	assertHasPair(t, wl, testpairs[1])

	if wl.Len() != 2 {
		t.Fatal("should have had two items")
	}

	if wl.Add(testpairs[1], 4, pb.Message_Wantlist_Block) {
		t.Fatal("add shouldnt report success on second add")
	}
	assertHasPair(t, wl, testpairs[0])
	assertHasPair(t, wl, testpairs[1])

	if wl.Len() != 2 {
		t.Fatal("should have had two items")
	}

	if !wl.RemoveType(testpairs[0], pb.Message_Wantlist_Block) {
		t.Fatal("should have gotten true")
	}

	assertHasPair(t, wl, testpairs[1])
	if _, has := wl.Contains(testpairs[0]); has {
		t.Fatal("shouldnt have this CidChannelPair")
	}
}

func TestAddHaveThenBlock(t *testing.T) {
	wl := New()

	wl.Add(testpairs[0], 5, pb.Message_Wantlist_Have)
	wl.Add(testpairs[0], 5, pb.Message_Wantlist_Block)

	e, ok := wl.Contains(testpairs[0])
	if !ok {
		t.Fatal("expected to have ", testpairs[0])
	}
	if e.WantType != pb.Message_Wantlist_Block {
		t.Fatal("expected to be ", pb.Message_Wantlist_Block)
	}
}

func TestAddBlockThenHave(t *testing.T) {
	wl := New()

	wl.Add(testpairs[0], 5, pb.Message_Wantlist_Block)
	wl.Add(testpairs[0], 5, pb.Message_Wantlist_Have)

	e, ok := wl.Contains(testpairs[0])
	if !ok {
		t.Fatal("expected to have ", testpairs[0])
	}
	if e.WantType != pb.Message_Wantlist_Block {
		t.Fatal("expected to be ", pb.Message_Wantlist_Block)
	}
}

func TestAddHaveThenRemoveBlock(t *testing.T) {
	wl := New()

	wl.Add(testpairs[0], 5, pb.Message_Wantlist_Have)
	wl.RemoveType(testpairs[0], pb.Message_Wantlist_Block)

	_, ok := wl.Contains(testpairs[0])
	if ok {
		t.Fatal("expected not to have ", testpairs[0])
	}
}

func TestAddBlockThenRemoveHave(t *testing.T) {
	wl := New()

	wl.Add(testpairs[0], 5, pb.Message_Wantlist_Block)
	wl.RemoveType(testpairs[0], pb.Message_Wantlist_Have)

	e, ok := wl.Contains(testpairs[0])
	if !ok {
		t.Fatal("expected to have ", testpairs[0])
	}
	if e.WantType != pb.Message_Wantlist_Block {
		t.Fatal("expected to be ", pb.Message_Wantlist_Block)
	}
}

func TestAddHaveThenRemoveAny(t *testing.T) {
	wl := New()

	wl.Add(testpairs[0], 5, pb.Message_Wantlist_Have)
	wl.Remove(testpairs[0])

	_, ok := wl.Contains(testpairs[0])
	if ok {
		t.Fatal("expected not to have ", testpairs[0])
	}
}

func TestAddBlockThenRemoveAny(t *testing.T) {
	wl := New()

	wl.Add(testpairs[0], 5, pb.Message_Wantlist_Block)
	wl.Remove(testpairs[0])

	_, ok := wl.Contains(testpairs[0])
	if ok {
		t.Fatal("expected not to have ", testpairs[0])
	}
}

func TestAbsort(t *testing.T) {
	wl := New()
	wl.Add(testpairs[0], 5, pb.Message_Wantlist_Block)
	wl.Add(testpairs[1], 4, pb.Message_Wantlist_Have)
	wl.Add(testpairs[2], 3, pb.Message_Wantlist_Have)

	wl2 := New()
	wl2.Add(testpairs[0], 2, pb.Message_Wantlist_Have)
	wl2.Add(testpairs[1], 1, pb.Message_Wantlist_Block)

	wl.Absorb(wl2)

	e, ok := wl.Contains(testpairs[0])
	if !ok {
		t.Fatal("expected to have ", testpairs[0])
	}
	if e.Priority != 5 {
		t.Fatal("expected priority 5")
	}
	if e.WantType != pb.Message_Wantlist_Block {
		t.Fatal("expected type ", pb.Message_Wantlist_Block)
	}

	e, ok = wl.Contains(testpairs[1])
	if !ok {
		t.Fatal("expected to have ", testpairs[1])
	}
	if e.Priority != 1 {
		t.Fatal("expected priority 1")
	}
	if e.WantType != pb.Message_Wantlist_Block {
		t.Fatal("expected type ", pb.Message_Wantlist_Block)
	}

	e, ok = wl.Contains(testpairs[2])
	if !ok {
		t.Fatal("expected to have ", testpairs[2])
	}
	if e.Priority != 3 {
		t.Fatal("expected priority 3")
	}
	if e.WantType != pb.Message_Wantlist_Have {
		t.Fatal("expected type ", pb.Message_Wantlist_Have)
	}
}

func TestSortEntries(t *testing.T) {
	wl := New()

	wl.Add(testpairs[0], 3, pb.Message_Wantlist_Block)
	wl.Add(testpairs[1], 5, pb.Message_Wantlist_Have)
	wl.Add(testpairs[2], 4, pb.Message_Wantlist_Have)

	entries := wl.Entries()
	SortEntries(entries)

	if !entries[0].Pair.Equals(testpairs[1]) ||
		!entries[1].Pair.Equals(testpairs[2]) ||
		!entries[2].Pair.Equals(testpairs[0]) {
		t.Fatal("wrong order")
	}
}
