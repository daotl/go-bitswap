package session

import (
	"testing"

	"github.com/daotl/go-bitswap/internal/testutil"
)

func TestSendWantBlocksTracker(t *testing.T) {
	peers := testutil.GeneratePeers(2)
	keys := testutil.GenerateWantKeys(2)
	swbt := newSentWantBlocksTracker()

	if swbt.haveSentWantBlockTo(peers[0], keys[0]) {
		t.Fatal("expected not to have sent anything yet")
	}

	swbt.addSentWantBlocksTo(peers[0], keys)
	if !swbt.haveSentWantBlockTo(peers[0], keys[0]) {
		t.Fatal("expected to have sent cid to peer")
	}
	if !swbt.haveSentWantBlockTo(peers[0], keys[1]) {
		t.Fatal("expected to have sent cid to peer")
	}
	if swbt.haveSentWantBlockTo(peers[1], keys[0]) {
		t.Fatal("expected not to have sent cid to peer")
	}
}
