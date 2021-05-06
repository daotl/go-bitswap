package wantlist

import (
	"github.com/ipfs/go-cid"
)

// Set is a implementation of a set of wantkeys, that is, a structure
// to which holds a single copy of every wantkeys that is added to it.
// note: not goroutine safe
type Set struct {
	set map[WantKey]struct{}
}

// NewSet initializes and returns a new Set.
func NewSet() *Set {
	return &Set{set: make(map[WantKey]struct{})}
}

func NewSetFromKeys(keys []WantKey) *Set {
	set := NewSet()
	for _, key := range keys {
		set.Add(key)
	}
	return set
}

// Add puts a WantKey in the Set.
func (s *Set) Add(c WantKey) {
	s.set[c] = struct{}{}
}

// Has returns if the Set contains a given Cid.
func (s *Set) Has(c WantKey) bool {
	_, ok := s.set[c]
	return ok
}

// Remove deletes a WantKey from the Set.
func (s *Set) Remove(c WantKey) {
	delete(s.set, c)
}

// Len returns how many elements the Set has.
func (s *Set) Len() int {
	return len(s.set)
}

// Keys returns the WantKeys in the set.
func (s *Set) Keys() []WantKey {
	out := make([]WantKey, 0, len(s.set))
	for k := range s.set {
		out = append(out, k)
	}
	return out
}

// Keys returns the WantKeys in the set.
func (s *Set) Cids() []cid.Cid {
	out := make([]cid.Cid, 0, len(s.set))
	for k := range s.set {
		out = append(out, k.Cid)
	}
	return out
}

// Visit adds a Cid to the set only if it is
// not in it already.
func (s *Set) Visit(c WantKey) bool {
	if !s.Has(c) {
		s.Add(c)
		return true
	}

	return false
}

// ForEach allows to run a custom function on each
// WantKey in the set.
func (s *Set) ForEach(f func(c WantKey) error) error {
	for c := range s.set {
		err := f(c)
		if err != nil {
			return err
		}
	}
	return nil
}
