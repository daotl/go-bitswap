package sessioninterestmanager

import (
	"sync"

	bsmsg "github.com/daotl/go-bitswap/message"
	wl "github.com/daotl/go-bitswap/wantlist"
)

// refactor: from cid.Cid to wl.WantKey

// SessionInterestManager records the keys that each session is interested in.
type SessionInterestManager struct {
	lk    sync.RWMutex
	wants map[wl.WantKey]map[uint64]bool
}

// New initializes a new SessionInterestManager.
func New() *SessionInterestManager {
	return &SessionInterestManager{
		// Map of wantkeys -> sessions -> bool
		//
		// The boolean indicates whether the session still wants the block
		// or is just interested in receiving messages about it.
		//
		// Note that once the block is received the session no longer wants
		// the block, but still wants to receive messages from peers who have
		// the block as they may have other blocks the session is interested in.
		wants: make(map[wl.WantKey]map[uint64]bool),
	}
}

// When the client asks the session for blocks, the session calls
// RecordSessionInterest() with those cids.
func (sim *SessionInterestManager) RecordSessionInterest(ses uint64, ks []wl.WantKey) {
	sim.lk.Lock()
	defer sim.lk.Unlock()

	// For each key
	for _, c := range ks {
		// Record that the session wants the blocks
		if want, ok := sim.wants[c]; ok {
			want[ses] = true
		} else {
			sim.wants[c] = map[uint64]bool{ses: true}
		}
	}
}

// When the session shuts down it calls RemoveSessionInterest().
// Returns the keys that no session is interested in any more.
func (sim *SessionInterestManager) RemoveSession(ses uint64) []wl.WantKey {
	sim.lk.Lock()
	defer sim.lk.Unlock()

	// The keys that no session is interested in
	deletedKs := make([]wl.WantKey, 0)

	// For each known key
	for c := range sim.wants {
		// Remove the session from the list of sessions that want the key
		delete(sim.wants[c], ses)

		// If there are no more sessions that want the key
		if len(sim.wants[c]) == 0 {
			// Clean up the list memory
			delete(sim.wants, c)
			// Add the key to the list of keys that no session is interested in
			deletedKs = append(deletedKs, c)
		}
	}

	return deletedKs
}

// When the session receives blocks, it calls RemoveSessionWants().
func (sim *SessionInterestManager) RemoveSessionWants(ses uint64, ks []wl.WantKey) {
	sim.lk.Lock()
	defer sim.lk.Unlock()

	// For each key
	for _, c := range ks {
		// If the session wanted the block
		if wanted, ok := sim.wants[c][ses]; ok && wanted {
			// Mark the block as unwanted
			sim.wants[c][ses] = false
		}
	}
}

// When a request is cancelled, the session calls RemoveSessionInterested().
// Returns the keys that no session is interested in any more.
func (sim *SessionInterestManager) RemoveSessionInterested(ses uint64, ks []wl.WantKey) []wl.WantKey {
	sim.lk.Lock()
	defer sim.lk.Unlock()

	// The keys that no session is interested in
	deletedKs := make([]wl.WantKey, 0, len(ks))

	// For each key
	for _, c := range ks {
		// If there is a list of sessions that want the key
		if _, ok := sim.wants[c]; ok {
			// Remove the session from the list of sessions that want the key
			delete(sim.wants[c], ses)

			// If there are no more sessions that want the key
			if len(sim.wants[c]) == 0 {
				// Clean up the list memory
				delete(sim.wants, c)
				// Add the key to the list of keys that no session is interested in
				deletedKs = append(deletedKs, c)
			}
		}
	}

	return deletedKs
}

// The session calls FilterSessionInterested() to filter the sets of keys for
// those that the session is interested in
func (sim *SessionInterestManager) FilterSessionInterested(ses uint64, ksets ...[]wl.WantKey) [][]wl.WantKey {
	sim.lk.RLock()
	defer sim.lk.RUnlock()

	// For each set of keys
	kres := make([][]wl.WantKey, len(ksets))
	for i, ks := range ksets {
		// The set of keys that at least one session is interested in
		has := make([]wl.WantKey, 0, len(ks))

		// For each key in the list
		for _, c := range ks {
			// If there is a session that's interested, add the key to the set
			if _, ok := sim.wants[c][ses]; ok {
				has = append(has, c)
			}
		}
		kres[i] = has
	}
	return kres
}

// When bitswap receives blocks it calls SplitWantedUnwanted() to discard
// unwanted blocks
func (sim *SessionInterestManager) SplitWantedUnwanted(blks []bsmsg.MsgBlock) ([]bsmsg.MsgBlock, []bsmsg.MsgBlock) {
	sim.lk.RLock()
	defer sim.lk.RUnlock()

	// Get the wanted block keys as a set
	wantedKs := wl.NewSet()
	for _, b := range blks {
		c := b.GetKey()
		// For each session that is interested in the key
		for ses := range sim.wants[c] {
			// If the session wants the key (rather than just being interested)
			if wanted, ok := sim.wants[c][ses]; ok && wanted {
				// Add the key to the set
				wantedKs.Add(c)
			}
		}
	}

	// Separate the blocks into wanted and unwanted
	wantedBlks := make([]bsmsg.MsgBlock, 0, len(blks))
	notWantedBlks := make([]bsmsg.MsgBlock, 0)
	for _, b := range blks {
		if wantedKs.Has(b.GetKey()) {
			wantedBlks = append(wantedBlks, b)
		} else {
			notWantedBlks = append(notWantedBlks, b)
		}
	}
	return wantedBlks, notWantedBlks
}

// When the SessionManager receives a message it calls InterestedSessions() to
// find out which sessions are interested in the message.
func (sim *SessionInterestManager) InterestedSessions(blks []wl.WantKey, haves []wl.WantKey, dontHaves []wl.WantKey) []uint64 {
	sim.lk.RLock()
	defer sim.lk.RUnlock()

	ks := make([]wl.WantKey, 0, len(blks)+len(haves)+len(dontHaves))
	ks = append(ks, blks...)
	ks = append(ks, haves...)
	ks = append(ks, dontHaves...)

	// Create a set of sessions that are interested in the keys
	sesSet := make(map[uint64]struct{})
	for _, c := range ks {
		for s := range sim.wants[c] {
			sesSet[s] = struct{}{}
		}
	}

	// Convert the set into a list
	ses := make([]uint64, 0, len(sesSet))
	for s := range sesSet {
		ses = append(ses, s)
	}
	return ses
}
