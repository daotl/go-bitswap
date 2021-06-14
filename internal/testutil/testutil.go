package testutil

import (
	"fmt"
	"math/rand"

	"github.com/daotl/go-ipld-channel/pair"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	blocksutil "github.com/ipfs/go-ipfs-blocksutil"
	"github.com/libp2p/go-libp2p-core/peer"

	bsmsg "github.com/daotl/go-bitswap/message"
	"github.com/daotl/go-bitswap/wantlist"
)

var blockGenerator = blocksutil.NewBlockGenerator()
var prioritySeq int32

// GenerateBlocksOfSize generates a series of blocks of the given byte size
func GenerateBlocksOfSize(n int, size int64) []blocks.Block {
	generatedBlocks := make([]blocks.Block, 0, n)
	for i := 0; i < n; i++ {
		// rand.Read never errors
		buf := make([]byte, size)
		rand.Read(buf)
		b := blocks.NewBlock(buf)
		generatedBlocks = append(generatedBlocks, b)

	}
	return generatedBlocks
}

// GenerateCids produces n content identifiers.
func GenerateCids(n int) []cid.Cid {
	cids := make([]cid.Cid, 0, n)
	for i := 0; i < n; i++ {
		c := blockGenerator.Next().Cid()
		cids = append(cids, c)
	}
	return cids
}

// GenerateCidChannelPairs produces n CidChannelPairs.
// TODO: GenerateCidChannelPairs should generate random IPLD channels
func GenerateCidChannelPairs(n int) []pair.CidChannelPair {
	pairs := make([]pair.CidChannelPair, 0, n)
	for i := 0; i < n; i++ {
		c := blockGenerator.Next().Cid()
		pairs = append(pairs, pair.PublicCidPair(c))
	}
	return pairs
}

// GenerateMessageEntries makes fake bitswap message entries.
func GenerateMessageEntries(n int, isCancel bool) []bsmsg.Entry {
	bsmsgs := make([]bsmsg.Entry, 0, n)
	for i := 0; i < n; i++ {
		prioritySeq++
		msg := bsmsg.Entry{
			Entry:  wantlist.NewRefEntry(pair.PublicCidPair(blockGenerator.Next().Cid()), prioritySeq),
			Cancel: isCancel,
		}
		bsmsgs = append(bsmsgs, msg)
	}
	return bsmsgs
}

var peerSeq int

// GeneratePeers creates n peer ids.
func GeneratePeers(n int) []peer.ID {
	peerIds := make([]peer.ID, 0, n)
	for i := 0; i < n; i++ {
		peerSeq++
		p := peer.ID(fmt.Sprint(i))
		peerIds = append(peerIds, p)
	}
	return peerIds
}

var nextSession uint64

// GenerateSessionID make a unit session identifier.
func GenerateSessionID() uint64 {
	nextSession++
	return uint64(nextSession)
}

// ContainsPeer returns true if a peer is found n a list of peers.
func ContainsPeer(peers []peer.ID, p peer.ID) bool {
	for _, n := range peers {
		if p == n {
			return true
		}
	}
	return false
}

// IndexOf returns the index of a given cid in an array of blocks
func IndexOf(blks []blocks.Block, c cid.Cid) int {
	for i, n := range blks {
		if n.Cid() == c {
			return i
		}
	}
	return -1
}

// ContainsBlock returns true if a block is found n a list of blocks
func ContainsBlock(blks []blocks.Block, block blocks.Block) bool {
	return IndexOf(blks, block.Cid()) != -1
}

// ContainsKey returns true if a key is found n a list of CIDs.
func ContainsKey(ks []cid.Cid, c cid.Cid) bool {
	for _, k := range ks {
		if c == k {
			return true
		}
	}
	return false
}

// MatchCidsIgnoreOrder returns true if the lists of CIDs match (even if
// they're in a different order)
func MatchCidsIgnoreOrder(cs1 []cid.Cid, cs2 []cid.Cid) bool {
	if len(cs1) != len(cs2) {
		return false
	}

	for _, k := range cs1 {
		if !ContainsKey(cs2, k) {
			return false
		}
	}
	return true
}

// MatchPairsIgnoreOrder returns true if the lists of CidChannelPairs match
// (even if they're in a different order)
func MatchPairsIgnoreOrder(ps1 []pair.CidChannelPair, ps2 []pair.CidChannelPair) bool {
	if len(ps1) != len(ps2) {
		return false
	}
	s2 := pair.NewSet()
	for _, key := range ps2 {
		s2.Add(key)
	}
	for _, k := range ps1 {
		if !s2.Has(k) {
			return false
		}
	}
	return true
}

// MatchPeersIgnoreOrder returns true if the lists of peers match (even if
// they're in a different order)
func MatchPeersIgnoreOrder(ps1 []peer.ID, ps2 []peer.ID) bool {
	if len(ps1) != len(ps2) {
		return false
	}

	for _, p := range ps1 {
		if !ContainsPeer(ps2, p) {
			return false
		}
	}
	return true
}
