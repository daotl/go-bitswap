package session

import (
	"fmt"
	"math/rand"
	"time"

	wl "github.com/daotl/go-bitswap/wantlist"
)

// liveWantsOrder and liveWants will get out of sync as blocks are received.
// This constant is the maximum amount to allow them to be out of sync before
// cleaning up the ordering array.
const liveWantsOrderGCLimit = 32

// sessionWants keeps track of which cids are waiting to be sent out, and which
// peers are "live" - ie, we've sent a request but haven't received a block yet
type sessionWants struct {
	// The wants that have not yet been sent out
	toFetch *wantKeyQueue
	// Wants that have been sent but have not received a response
	liveWants map[wl.WantKey]time.Time
	// The order in which wants were requested
	liveWantsOrder []wl.WantKey
	// The maximum number of want-haves to send in a broadcast
	broadcastLimit int
}

func newSessionWants(broadcastLimit int) sessionWants {
	return sessionWants{
		toFetch:        newKeyQueue(),
		liveWants:      make(map[wl.WantKey]time.Time),
		broadcastLimit: broadcastLimit,
	}
}

func (sw *sessionWants) String() string {
	return fmt.Sprintf("%d pending / %d live", sw.toFetch.Len(), len(sw.liveWants))
}

// BlocksRequested is called when the client makes a request for blocks
func (sw *sessionWants) BlocksRequested(newWants []wl.WantKey) {
	for _, k := range newWants {
		sw.toFetch.Push(k)
	}
}

// GetNextWants is called when the session has not yet discovered peers with
// the blocks that it wants. It moves as many CIDs from the fetch queue to
// the live wants queue as possible (given the broadcast limit).
// Returns the newly live wants.
func (sw *sessionWants) GetNextWants() []wl.WantKey {
	now := time.Now()

	// Move CIDs from fetch queue to the live wants queue (up to the broadcast
	// limit)
	currentLiveCount := len(sw.liveWants)
	toAdd := sw.broadcastLimit - currentLiveCount

	var live []wl.WantKey
	for ; toAdd > 0 && sw.toFetch.Len() > 0; toAdd-- {
		c := sw.toFetch.Pop()
		live = append(live, c)
		sw.liveWantsOrder = append(sw.liveWantsOrder, c)
		sw.liveWants[c] = now
	}

	return live
}

// WantsSent is called when wants are sent to a peer
func (sw *sessionWants) WantsSent(ks []wl.WantKey) {
	now := time.Now()
	for _, c := range ks {
		if _, ok := sw.liveWants[c]; !ok && sw.toFetch.Has(c) {
			sw.toFetch.Remove(c)
			sw.liveWantsOrder = append(sw.liveWantsOrder, c)
			sw.liveWants[c] = now
		}
	}
}

// BlocksReceived removes received block CIDs from the live wants list and
// measures latency. It returns the CIDs of blocks that were actually
// wanted (as opposed to duplicates) and the total latency for all incoming blocks.
func (sw *sessionWants) BlocksReceived(ks []wl.WantKey) ([]wl.WantKey, time.Duration) {
	wanted := make([]wl.WantKey, 0, len(ks))
	totalLatency := time.Duration(0)
	if len(ks) == 0 {
		return wanted, totalLatency
	}

	// Filter for blocks that were actually wanted (as opposed to duplicates)
	now := time.Now()
	for _, c := range ks {
		if sw.isWanted(c) {
			wanted = append(wanted, c)

			// Measure latency
			sentAt, ok := sw.liveWants[c]
			if ok && !sentAt.IsZero() {
				totalLatency += now.Sub(sentAt)
			}

			// Remove the CID from the live wants / toFetch queue
			delete(sw.liveWants, c)
			sw.toFetch.Remove(c)
		}
	}

	// If the live wants ordering array is a long way out of sync with the
	// live wants map, clean up the ordering array
	if len(sw.liveWantsOrder)-len(sw.liveWants) > liveWantsOrderGCLimit {
		cleaned := sw.liveWantsOrder[:0]
		for _, c := range sw.liveWantsOrder {
			if _, ok := sw.liveWants[c]; ok {
				cleaned = append(cleaned, c)
			}
		}
		sw.liveWantsOrder = cleaned
	}

	return wanted, totalLatency
}

// PrepareBroadcast saves the current time for each live want and returns the
// live want CIDs up to the broadcast limit.
func (sw *sessionWants) PrepareBroadcast() []wl.WantKey {
	now := time.Now()
	live := make([]wl.WantKey, 0, len(sw.liveWants))
	for _, c := range sw.liveWantsOrder {
		if _, ok := sw.liveWants[c]; ok {
			// No response was received for the want, so reset the sent time
			// to now as we're about to broadcast
			sw.liveWants[c] = now

			live = append(live, c)
			if len(live) == sw.broadcastLimit {
				break
			}
		}
	}

	return live
}

// CancelPending removes the given CIDs from the fetch queue.
func (sw *sessionWants) CancelPending(keys []wl.WantKey) {
	for _, k := range keys {
		sw.toFetch.Remove(k)
	}
}

// LiveWants returns a list of live wants
func (sw *sessionWants) LiveWants() []wl.WantKey {
	live := make([]wl.WantKey, 0, len(sw.liveWants))
	for c := range sw.liveWants {
		live = append(live, c)
	}

	return live
}

// RandomLiveWant returns a randomly selected live want
func (sw *sessionWants) RandomLiveWant() wl.WantKey {
	if len(sw.liveWants) == 0 {
		return wl.WantKey{}
	}

	// picking a random live want
	i := rand.Intn(len(sw.liveWants))
	for k := range sw.liveWants {
		if i == 0 {
			return k
		}
		i--
	}
	return wl.WantKey{}
}

// Has live wants indicates if there are any live wants
func (sw *sessionWants) HasLiveWants() bool {
	return len(sw.liveWants) > 0
}

// Indicates whether the want is in either of the fetch or live queues
func (sw *sessionWants) isWanted(c wl.WantKey) bool {
	_, ok := sw.liveWants[c]
	if !ok {
		ok = sw.toFetch.Has(c)
	}
	return ok
}
