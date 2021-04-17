package channel

import (
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
)

type Id cid.Cid

// Filter return true if the peer and cid belong to this channel
// Peers in the channel have access to cids in the channel
type Filter func(pid peer.ID, id cid.Cid, channel Id) (bool, error)
