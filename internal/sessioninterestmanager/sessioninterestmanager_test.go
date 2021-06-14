package sessioninterestmanager

import (
	"testing"

	"github.com/daotl/go-ipld-channel/pair"
	"github.com/ipfs/go-cid"

	"github.com/daotl/go-bitswap/internal/testutil"
)

func TestEmpty(t *testing.T) {
	sim := New()

	ses := uint64(1)
	blks := testutil.GenerateCids(2)
	// TODO: GenerateCidChannelPairs should generate random IPLD channels
	haves := testutil.GenerateCidChannelPairs(2)
	dontHaves := testutil.GenerateCidChannelPairs(2)
	filteredBlks, filteredHaves, filteredDontHaves :=
		sim.FilterSessionInterested(ses, blks, haves, dontHaves)
	if len(filteredBlks) > 0 || len(filteredHaves) > 0 || len(filteredDontHaves) > 0 {
		t.Fatal("Expected no interest")
	}
	if len(sim.InterestedSessions(blks, haves, dontHaves)) > 0 {
		t.Fatal("Expected no interest")
	}
}

func TestBasic(t *testing.T) {
	sim := New()

	ses1 := uint64(1)
	ses2 := uint64(2)
	blks := testutil.GenerateCids(2)
	haves := testutil.GenerateCidChannelPairs(2)
	dontHaves := testutil.GenerateCidChannelPairs(2)
	pairs1 := testutil.GenerateCidChannelPairs(2)
	pairs2 := append(testutil.GenerateCidChannelPairs(1), pairs1[1])
	cids1 := pair.PairsToCids(pairs1)
	cids2 := pair.PairsToCids(pairs2)

	// ---

	sim.RecordSessionInterest(ses1, pairs1)

	filteredBlks, _, _ := sim.FilterSessionInterested(ses1, cids1, haves, dontHaves)
	if len(filteredBlks) != 2 {
		t.Fatal("Expected 2 keys")
	}
	if len(sim.InterestedSessions(cids1, haves, dontHaves)) != 1 {
		t.Fatal("Expected 1 session")
	}

	_, filteredHaves, _ := sim.FilterSessionInterested(ses1, blks, pairs1, dontHaves)
	if len(filteredHaves) != 2 {
		t.Fatal("Expected 2 keys")
	}
	if len(sim.InterestedSessions(blks, pairs1, dontHaves)) != 1 {
		t.Fatal("Expected 1 session")
	}

	_, _, filteredDontHaves := sim.FilterSessionInterested(ses1, blks, haves, pairs1)
	if len(filteredDontHaves) != 2 {
		t.Fatal("Expected 2 keys")
	}
	if len(sim.InterestedSessions(blks, haves, pairs1)) != 1 {
		t.Fatal("Expected 1 session")
	}

	// ---

	sim.RecordSessionInterest(ses2, pairs2)

	filteredBlks, _, _ = sim.FilterSessionInterested(ses2, cids1[:1], haves, dontHaves)
	if len(filteredBlks) != 0 {
		t.Fatal("Expected no interest")
	}
	filteredBlks, _, _ = sim.FilterSessionInterested(ses2, cids2, haves, dontHaves)
	if len(filteredBlks) != 2 {
		t.Fatal("Expected 2 keys")
	}
	if len(sim.InterestedSessions(cids1[:1], haves, dontHaves)) != 1 {
		t.Fatal("Expected 1 session")
	}
	if len(sim.InterestedSessions(cids1[1:], haves, dontHaves)) != 2 {
		t.Fatal("Expected 2 sessions")
	}

	_, filteredHaves, _ = sim.FilterSessionInterested(ses2, blks, pairs1[:1], dontHaves)
	if len(filteredHaves) != 0 {
		t.Fatal("Expected no interest")
	}
	_, filteredHaves, _ = sim.FilterSessionInterested(ses2, blks, pairs2, dontHaves)
	if len(filteredHaves) != 2 {
		t.Fatal("Expected 2 keys")
	}
	if len(sim.InterestedSessions(blks, pairs1[:1], dontHaves)) != 1 {
		t.Fatal("Expected 1 session")
	}
	if len(sim.InterestedSessions(blks, pairs1[1:], dontHaves)) != 2 {
		t.Fatal("Expected 2 sessions")
	}

	sim.RecordSessionInterest(ses2, pairs2)
	_, _, filteredDontHaves = sim.FilterSessionInterested(ses2, blks, haves, pairs1[:1])
	if len(filteredDontHaves) != 0 {
		t.Fatal("Expected no interest")
	}
	_, _, filteredDontHaves = sim.FilterSessionInterested(ses2, blks, haves, pairs2)
	if len(filteredDontHaves) != 2 {
		t.Fatal("Expected 2 keys")
	}
	if len(sim.InterestedSessions(blks, haves, pairs1[:1])) != 1 {
		t.Fatal("Expected 1 session")
	}
	if len(sim.InterestedSessions(blks, haves, pairs1[1:])) != 2 {
		t.Fatal("Expected 2 sessions")
	}
}

func TestInterestedSessions(t *testing.T) {
	sim := New()

	ses := uint64(1)
	emptyPairs := []pair.CidChannelPair{}
	emptyCids := []cid.Cid{}
	pairs := testutil.GenerateCidChannelPairs(3)
	cids := pair.PairsToCids(pairs)
	sim.RecordSessionInterest(ses, pairs[0:2])

	if len(sim.InterestedSessions(cids, emptyPairs, emptyPairs)) != 1 {
		t.Fatal("Expected 1 session")
	}
	if len(sim.InterestedSessions(cids[0:1], emptyPairs, emptyPairs)) != 1 {
		t.Fatal("Expected 1 session")
	}
	if len(sim.InterestedSessions(emptyCids, pairs, emptyPairs)) != 1 {
		t.Fatal("Expected 1 session")
	}
	if len(sim.InterestedSessions(emptyCids, pairs[0:1], emptyPairs)) != 1 {
		t.Fatal("Expected 1 session")
	}
	if len(sim.InterestedSessions(emptyCids, emptyPairs, pairs)) != 1 {
		t.Fatal("Expected 1 session")
	}
	if len(sim.InterestedSessions(emptyCids, emptyPairs, pairs[0:1])) != 1 {
		t.Fatal("Expected 1 session")
	}
}

func TestRemoveSession(t *testing.T) {
	sim := New()

	ses1 := uint64(1)
	ses2 := uint64(2)
	cids := testutil.GenerateCids(2)
	pairs1 := testutil.GenerateCidChannelPairs(2)
	pairs2 := append(testutil.GenerateCidChannelPairs(1), pairs1[1])
	sim.RecordSessionInterest(ses1, pairs1)
	sim.RecordSessionInterest(ses2, pairs2)
	sim.RemoveSession(ses1)

	_, res, _ := sim.FilterSessionInterested(ses1, cids, pairs1, []pair.CidChannelPair{})
	if len(res) != 0 {
		t.Fatal("Expected no interest")
	}

	_, res2, res3 := sim.FilterSessionInterested(ses2, cids, pairs1, pairs2)
	if len(res2) != 1 {
		t.Fatal("Expected 1 key")
	}
	if len(res3) != 2 {
		t.Fatal("Expected 2 keys")
	}
}

func TestRemoveSessionInterested(t *testing.T) {
	sim := New()

	ses1 := uint64(1)
	ses2 := uint64(2)
	emptyPairs := []pair.CidChannelPair{}
	cids := testutil.GenerateCids(2)
	pairs1 := testutil.GenerateCidChannelPairs(2)
	pairs2 := append(testutil.GenerateCidChannelPairs(1), pairs1[1])
	sim.RecordSessionInterest(ses1, pairs1)
	sim.RecordSessionInterest(ses2, pairs2)

	res := sim.RemoveSessionInterested(ses1, []cid.Cid{pairs1[0].Cid})
	if len(res) != 1 {
		t.Fatal("Expected no interested sessions left")
	}

	_, interested, _ := sim.FilterSessionInterested(ses1, cids, pairs1, emptyPairs)
	if len(interested) != 1 {
		t.Fatal("Expected ses1 still interested in one cid")
	}

	res = sim.RemoveSessionInterested(ses1, pair.PairsToCids(pairs1))
	if len(res) != 0 {
		t.Fatal("Expected ses2 to be interested in one cid")
	}

	_, interested, _ = sim.FilterSessionInterested(ses1, cids, pairs1, emptyPairs)
	if len(interested) != 0 {
		t.Fatal("Expected ses1 to have no remaining interest")
	}

	_, interested, _ = sim.FilterSessionInterested(ses2, cids, pairs1, emptyPairs)
	if len(interested) != 1 {
		t.Fatal("Expected ses2 to still be interested in one key")
	}
}

func TestSplitWantedUnwanted(t *testing.T) {
	blks := testutil.GenerateBlocksOfSize(3, 1024)
	sim := New()
	ses1 := uint64(1)
	ses2 := uint64(2)

	var pairs []pair.CidChannelPair
	for _, b := range blks {
		pairs = append(pairs, pair.PublicCidPair(b.Cid()))
	}

	// ses1: <none>
	// ses2: <none>
	wanted, unwanted := sim.SplitWantedUnwanted(blks)
	if len(wanted) > 0 {
		t.Fatal("Expected no blocks")
	}
	if len(unwanted) != 3 {
		t.Fatal("Expected 3 blocks")
	}

	// ses1: 0 1
	// ses2: <none>
	sim.RecordSessionInterest(ses1, pairs[0:2])
	wanted, unwanted = sim.SplitWantedUnwanted(blks)
	if len(wanted) != 2 {
		t.Fatal("Expected 2 blocks")
	}
	if len(unwanted) != 1 {
		t.Fatal("Expected 1 block")
	}

	// ses1: 1
	// ses2: 1 2
	sim.RecordSessionInterest(ses2, pairs[1:])
	sim.RemoveSessionWants(ses1, pair.PairsToCids(pairs[:1]))

	wanted, unwanted = sim.SplitWantedUnwanted(blks)
	if len(wanted) != 2 {
		t.Fatal("Expected 2 blocks")
	}
	if len(unwanted) != 1 {
		t.Fatal("Expected no blocks")
	}

	// ses1: <none>
	// ses2: 1 2
	sim.RemoveSessionWants(ses1, pair.PairsToCids(pairs[1:2]))

	wanted, unwanted = sim.SplitWantedUnwanted(blks)
	if len(wanted) != 2 {
		t.Fatal("Expected 2 blocks")
	}
	if len(unwanted) != 1 {
		t.Fatal("Expected no blocks")
	}

	// ses1: <none>
	// ses2: 2
	sim.RemoveSessionWants(ses2, pair.PairsToCids(pairs[1:2]))

	wanted, unwanted = sim.SplitWantedUnwanted(blks)
	if len(wanted) != 1 {
		t.Fatal("Expected 2 blocks")
	}
	if len(unwanted) != 2 {
		t.Fatal("Expected 2 blocks")
	}
}
