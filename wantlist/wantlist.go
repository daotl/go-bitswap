// Package wantlist implements an object for bitswap that contains the keys
// that a given peer wants.
package wantlist

import (
	"sort"

	pb "github.com/daotl/go-bitswap/message/pb"
	"github.com/daotl/go-ipld-channel/pair"
)

// Wantlist is a raw list of wanted blocks and their priorities
type Wantlist struct {
	set map[pair.CidChannelPair]Entry
}

// Entry is an entry in a want list, consisting of a CidChannelPair and its priority
type Entry struct {
	Pair     pair.CidChannelPair
	Priority int32
	WantType pb.Message_Wantlist_WantType
}

// NewRefEntry creates a new reference tracked wantlist entry.
func NewRefEntry(ccp pair.CidChannelPair, p int32) Entry {
	return Entry{
		Pair:     ccp,
		Priority: p,
		WantType: pb.Message_Wantlist_Block,
	}
}

type entrySlice []Entry

func (es entrySlice) Len() int           { return len(es) }
func (es entrySlice) Swap(i, j int)      { es[i], es[j] = es[j], es[i] }
func (es entrySlice) Less(i, j int) bool { return es[i].Priority > es[j].Priority }

// New generates a new raw Wantlist
func New() *Wantlist {
	return &Wantlist{
		set: make(map[pair.CidChannelPair]Entry),
	}
}

// Len returns the number of entries in a wantlist.
func (w *Wantlist) Len() int {
	return len(w.set)
}

// Add adds an entry in a wantlist from CidChannelPair & Priority, if not already present.
func (w *Wantlist) Add(pair pair.CidChannelPair, priority int32,
	wantType pb.Message_Wantlist_WantType) bool {
	e, ok := w.set[pair]

	// Adding want-have should not override want-block
	if ok && (e.WantType == pb.Message_Wantlist_Block || wantType == pb.Message_Wantlist_Have) {
		return false
	}

	w.set[pair] = Entry{
		Pair:     pair,
		Priority: priority,
		WantType: wantType,
	}

	return true
}

// Remove removes the given CidChannelPair from the wantlist.
func (w *Wantlist) Remove(p pair.CidChannelPair) bool {
	_, ok := w.set[p]
	if !ok {
		return false
	}

	delete(w.set, p)
	return true
}

// RemoveType removes the given CidChannelPair from the wantlist, respecting the type:
// Remove with want-have will not remove an existing want-block.
func (w *Wantlist) RemoveType(p pair.CidChannelPair, wantType pb.Message_Wantlist_WantType) bool {
	e, ok := w.set[p]
	if !ok {
		return false
	}

	// Removing want-have should not remove want-block
	if e.WantType == pb.Message_Wantlist_Block && wantType == pb.Message_Wantlist_Have {
		return false
	}

	delete(w.set, p)
	return true
}

// Contains returns the entry, if present, for the given CidChannelPair, plus whether
// it was present.
func (w *Wantlist) Contains(p pair.CidChannelPair) (Entry, bool) {
	e, ok := w.set[p]
	return e, ok
}

// Entries returns all wantlist entries for a want list.
func (w *Wantlist) Entries() []Entry {
	es := make([]Entry, 0, len(w.set))
	for _, e := range w.set {
		es = append(es, e)
	}
	return es
}

// Absorb all the entries in other into this want list
func (w *Wantlist) Absorb(other *Wantlist) {
	for _, e := range other.Entries() {
		w.Add(e.Pair, e.Priority, e.WantType)
	}
}

// SortEntries sorts the list of entries by priority.
func SortEntries(es []Entry) {
	sort.Sort(entrySlice(es))
}
