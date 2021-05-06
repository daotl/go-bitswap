package accesscontrol

import (
	"errors"
	"fmt"

	wl "github.com/daotl/go-bitswap/wantlist"
	exchange "github.com/daotl/go-ipfs-exchange-interface"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
)

// ACFilter should return whether the peer has permission to access the specified
// block in the specified exchange channel.
type ACFilter func(pid peer.ID, channel exchange.Channel, id cid.Cid) (bool, error)

var GlobalFilter ACFilter = PassAll

func PassAll(pid peer.ID, channel exchange.Channel, id cid.Cid) (bool, error) {
	return true, nil
}

func ResetFilter() {
	GlobalFilter = PassAll
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

func FilterKeys(keys []wl.WantKey, id peer.ID) []wl.WantKey {
	set := wl.NewSet()
	for _, c := range keys {
		ok, _ := GlobalFilter(id, c.Ch, c.Cid)
		if ok {
			set.Add(c)
		}
	}
	return set.Keys()
}

func FilterCids(keys []cid.Cid, ch exchange.Channel, id peer.ID) []cid.Cid {
	set := cid.NewSet()
	for _, c := range keys {
		ok, _ := GlobalFilter(id, ch, c)
		if ok {
			set.Add(c)
		}
	}
	return set.Keys()
}

// channel `ch` includes peers specified by `pids` and data specified by `cids`
func FixedChannel(ch exchange.Channel, pids []peer.ID, cids []cid.Cid) ACFilter {
	cset := cid.NewSet()
	for _, id := range cids {
		cset.Add(id)
	}
	pset := peer.NewSet()
	for _, pid := range pids {
		pset.Add(pid)
	}
	return func(pid peer.ID, channel exchange.Channel, id cid.Cid) (bool, error) {
		if channel != ch {
			return false, errors.New("channel mismatch")
		}
		if !cset.Has(id) {
			return false, fmt.Errorf("cid %v is not in this channel", id)
		}
		if !pset.Contains(pid) {
			return false, fmt.Errorf("peerid %v is not in this channel", pid)
		}
		return true, nil
	}
}

/*
HookAfterIssueRequest

HookBeforeSendWant

HookAfterReceiveWant

HookBeforeSendResponse

HookAfterReceiveBlock
*/
