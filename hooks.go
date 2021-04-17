package bitswap

import (
	"github.com/daotl/go-bitswap/channel"
)

func WithFilter(filter channel.Filter) Option {
	return func(bs *Bitswap) {
		// TODO:
	}
}

/*
HookAfterIssueRequest

HookBeforeSendWant

HookAfterReceiveWant

HookBeforeSendResponse

HookAfterReceiveBlock
*/
