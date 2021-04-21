package accesscontrol

import (
	"fmt"

	exchange "github.com/daotl/go-ipfs-exchange-interface"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
)

// ACFilter should return whether the peer has permission to access the specified
// block in the specified exchange channel.
// if id == cid.Undef, it checks whether the peer pid is in the channel
type ACFilter func(pid peer.ID, channel exchange.Channel, id cid.Cid) (bool, error)

var GlobalFilter ACFilter = PassAll

func PassAll(pid peer.ID, channel exchange.Channel, id cid.Cid) (bool, error) {
	return true, nil
}

func WrapError(filter ACFilter) ACFilter {
	return func(pid peer.ID, channel exchange.Channel, id cid.Cid) (bool, error) {
		ok, err := filter(pid, channel, id)
		if ok {
			return true, nil
		} else {
			return false, ErrBlocked(pid, channel, id, err)
		}
	}
}

func ErrBlocked(pid peer.ID, ch exchange.Channel, k cid.Cid, err error) error {
	return fmt.Errorf("peer %s does not have access to cid %s in channel %v: %v", pid.Pretty(), k.String(), ch, err)
}

/*
HookAfterIssueRequest

HookBeforeSendWant

HookAfterReceiveWant

HookBeforeSendResponse

HookAfterReceiveBlock
*/
