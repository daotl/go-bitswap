package session

import (
	wl "github.com/daotl/go-bitswap/wantlist"
)

// a queue with distinct elements(wantKeys), supporting access by key besides pushing and popping.
// not routine-safe
type wantKeyQueue struct {
	elems []wl.WantKey
	eset  *wl.Set
}

func newKeyQueue() *wantKeyQueue {
	return &wantKeyQueue{eset: wl.NewSet()}
}

func (cq *wantKeyQueue) Pop() wl.WantKey {
	for {
		if len(cq.elems) == 0 {
			return wl.WantKey{}
		}

		out := cq.elems[0]
		cq.elems = cq.elems[1:]

		if cq.eset.Has(out) {
			cq.eset.Remove(out)
			return out
		}
	}
}

func (cq *wantKeyQueue) Cids() []wl.WantKey {
	// Lazily delete from the list any cids that were removed from the set
	if len(cq.elems) > cq.eset.Len() {
		i := 0
		for _, c := range cq.elems {
			if cq.eset.Has(c) {
				cq.elems[i] = c
				i++
			}
		}
		cq.elems = cq.elems[:i]
	}

	// Make a copy of the cids
	return append([]wl.WantKey{}, cq.elems...)
}

func (cq *wantKeyQueue) Push(c wl.WantKey) {
	if cq.eset.Visit(c) {
		cq.elems = append(cq.elems, c)
	}
}

func (cq *wantKeyQueue) Remove(c wl.WantKey) {
	cq.eset.Remove(c)
}

func (cq *wantKeyQueue) Has(c wl.WantKey) bool {
	return cq.eset.Has(c)
}

func (cq *wantKeyQueue) Len() int {
	return cq.eset.Len()
}
