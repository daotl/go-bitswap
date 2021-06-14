package sessioninterestmanager

import (
	"sync"

	channel "github.com/daotl/go-ipld-channel"
	"github.com/daotl/go-ipld-channel/pair"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
)

// SessionInterestManager records the CIDs and IPLD Channels that each session is interested in.
type SessionInterestManager struct {
	lk    sync.RWMutex
	wants map[cid.Cid]sessionChannels
}

type sessionChannels = map[uint64]map[channel.Channel]bool

// New initializes a new SessionInterestManager.
func New() *SessionInterestManager {
	return &SessionInterestManager{
		// Map of cids -> sessions -> IPLD channels -> bool
		//
		// The boolean indicates whether the session still wants the block
		// or is just interested in receiving messages about it.
		//
		// Note that once the block is received the session no longer wants
		// the block, but still wants to receive messages from peers who have
		// the block as they may have other blocks the session is interested in.
		wants: make(map[cid.Cid]sessionChannels),
	}
}

// When the client asks the session for blocks, the session calls
// RecordSessionInterest() with those CidChannelPairs.
func (sim *SessionInterestManager) RecordSessionInterest(ses uint64, ks []pair.CidChannelPair) {
	sim.lk.Lock()
	defer sim.lk.Unlock()

	// For each CidChannelPair
	for _, p := range ks {
		// Record that the session wants the blocks
		want, ok := sim.wants[p.Cid]
		if !ok {
			want = make(sessionChannels)
			sim.wants[p.Cid] = want
		}
		chnmap, ok := want[ses]
		if !ok {
			chnmap = make(map[channel.Channel]bool)
			want[ses] = chnmap
		}
		chnmap[p.Channel] = true
	}
}

// When the session shuts down it calls RemoveSessionInterest().
// Returns the CIDs that no session is interested in any more.
func (sim *SessionInterestManager) RemoveSession(ses uint64) []cid.Cid {
	sim.lk.Lock()
	defer sim.lk.Unlock()

	// The CIDs that no session is interested in
	deletedKs := make([]cid.Cid, 0)

	// For each known key
	for c := range sim.wants {
		// Remove the session from the list of sessions that want the CIDs
		delete(sim.wants[c], ses)

		// If there are no more sessions that want the CID
		if len(sim.wants[c]) == 0 {
			// Clean up the list memory
			delete(sim.wants, c)
			// Add the CID to the list of CIDs that no session is interested in
			deletedKs = append(deletedKs, c)
		}
	}

	return deletedKs
}

// When the session receives blocks, it calls RemoveSessionWants().
func (sim *SessionInterestManager) RemoveSessionWants(ses uint64, ks []cid.Cid) {
	sim.lk.Lock()
	defer sim.lk.Unlock()

	// For each CID
	for _, c := range ks {
		// If the session wanted the block
		if chnmap, ok := sim.wants[c][ses]; ok {
			// For each IPLD channel
			for chn := range chnmap {
				// Mark the block as unwanted
				chnmap[chn] = false
			}

		}
	}
}

// When a request is cancelled, the session calls RemoveSessionInterested().
// Returns the CIDs that no session is interested in any more.
func (sim *SessionInterestManager) RemoveSessionInterested(ses uint64, ks []cid.Cid) []cid.Cid {
	sim.lk.Lock()
	defer sim.lk.Unlock()

	// The CID that no session is interested in
	deletedKs := make([]cid.Cid, 0, len(ks))

	// For each CID
	for _, c := range ks {
		// If there is a list of sessions that are interested in the CID,
		// or has a empty sessionChannels
		if _, ok := sim.wants[c]; ok {
			// Remove the session from the list of sessions that are interested in the CID
			delete(sim.wants[c], ses)

			// If there are no more sessions that are interested in the CID,
			// or has an empty sessionChannels
			if len(sim.wants[c]) == 0 {
				// Clean up the list memory
				delete(sim.wants, c)
				// Add the CID to the list of CIDs that no session is interested in
				deletedKs = append(deletedKs, c)
			}
		}
	}

	return deletedKs
}

// The session calls FilterSessionInterested() to filter the sets of keys for
// those that the session is interested in
func (sim *SessionInterestManager) FilterSessionInterested(ses uint64, blks []cid.Cid,
	haves []pair.CidChannelPair, dontHaves []pair.CidChannelPair) (filteredBlks []cid.Cid,
	filteredHaves []pair.CidChannelPair, filteredDontHaves []pair.CidChannelPair) {
	sim.lk.RLock()
	defer sim.lk.RUnlock()

	// Filter blks
	// The set of CIDs that at least one session is interested in
	filteredBlks = make([]cid.Cid, 0, len(blks))
	// For each CID in the list
	for _, c := range blks {
		// If there is a session that's interested, add the CID to the filtered list
		if chnmap, ok := sim.wants[c][ses]; ok && len(chnmap) > 0 {
			filteredBlks = append(filteredBlks, c)
		}
	}

	// Filter haves and dontHaves
	kres := make([][]pair.CidChannelPair, 2)
	for i, ks := range [][]pair.CidChannelPair{haves, dontHaves} {
		// The set of CidChannelPairs that at least one session is interested in
		has := make([]pair.CidChannelPair, 0, len(ks))

		// For each CidChannelPair in the list
		for _, p := range ks {
			// If there is a session that's interested, add the CidChannelPair to the filtered list
			if _, ok := sim.wants[p.Cid][ses][p.Channel]; ok {
				has = append(has, p)
			}
		}
		kres[i] = has
	}
	return filteredBlks, kres[0], kres[1]
}

// When bitswap receives blocks it calls SplitWantedUnwanted() to discard
// unwanted blocks
func (sim *SessionInterestManager) SplitWantedUnwanted(blks []blocks.Block) ([]blocks.Block, []blocks.Block) {
	sim.lk.RLock()
	defer sim.lk.RUnlock()

	// Get the wanted block CIDs as a set
	wantedKs := cid.NewSet()
	for _, b := range blks {
		c := b.Cid()
	Sessions:
		// For each session that is interested in the CID
		for _, chnmap := range sim.wants[c] {
			// For each IPLD channel
			for _, wanted := range chnmap {
				// If the session wants the CID (rather than just being interested)
				if wanted {
					// Add the CID to the set
					wantedKs.Add(c)
					// Only need to add each CID once
					break Sessions
				}
			}
		}
	}

	// Separate the blocks into wanted and unwanted
	wantedBlks := make([]blocks.Block, 0, len(blks))
	notWantedBlks := make([]blocks.Block, 0)
	for _, b := range blks {
		if wantedKs.Has(b.Cid()) {
			wantedBlks = append(wantedBlks, b)
		} else {
			notWantedBlks = append(notWantedBlks, b)
		}
	}
	return wantedBlks, notWantedBlks
}

// When the SessionManager receives a message it calls InterestedSessions() to
// find out which sessions are interested in the message.
func (sim *SessionInterestManager) InterestedSessions(blks []cid.Cid,
	haves []pair.CidChannelPair, dontHaves []pair.CidChannelPair) []uint64 {
	sim.lk.RLock()
	defer sim.lk.RUnlock()

	// Create a set of sessions that are interested in the message
	sesSet := make(map[uint64]struct{})

	// Add to the set the sessions that are interested in the CIDs in blks
	for _, c := range blks {
		for s, chnmap := range sim.wants[c] {
			if len(chnmap) > 0 {
				sesSet[s] = struct{}{}
			}
		}
	}

	// Add to the set the sessions that are interested in the CID and
	// IPLD channel pairs in haves and dontHaves
	ks := make([]pair.CidChannelPair, 0, len(haves)+len(dontHaves))
	ks = append(ks, haves...)
	ks = append(ks, dontHaves...)
	for _, p := range ks {
		for s, chnmap := range sim.wants[p.Cid] {
			if _, ok := chnmap[p.Channel]; ok {
				sesSet[s] = struct{}{}
			}
		}
	}

	// Convert the set into a list
	ses := make([]uint64, 0, len(sesSet))
	for s := range sesSet {
		ses = append(ses, s)
	}
	return ses
}
