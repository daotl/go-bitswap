package accesscontrol

import (
	exchange "github.com/daotl/go-ipfs-exchange-interface"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
)

// ACFilter should return whether the peer has permission to access the specified
// block in the specified exchange channel.
type ACFilter func(pid peer.ID, channel exchange.Channel, id cid.Cid) (bool, error)

/*
HookAfterIssueRequest

HookBeforeSendWant

HookAfterReceiveWant

HookBeforeSendResponse

HookAfterReceiveBlock
*/
