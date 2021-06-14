package notifications

import (
	"context"
	"sync"

	"github.com/cskr/pubsub"
	"github.com/daotl/go-bitswap/message"
	wl "github.com/daotl/go-bitswap/wantlist"
)

const bufferSize = 16

// PubSub is a simple interface for publishing blocks and being able to subscribe
// for cids. It's used internally by bitswap to decouple receiving blocks
// and actually providing them back to the GetBlocks caller.
type PubSub interface {
	Publish(block message.MsgBlock)
	Subscribe(ctx context.Context, keys ...wl.WantKey) <-chan message.MsgBlock
	Shutdown()
}

// New generates a new PubSub interface.
func New() PubSub {
	return &impl{
		wrapped: *pubsub.New(bufferSize),
		closed:  make(chan struct{}),
	}
}

type impl struct {
	lk      sync.RWMutex
	wrapped pubsub.PubSub

	closed chan struct{}
}

func (ps *impl) Publish(block message.MsgBlock) {
	ps.lk.RLock()
	defer ps.lk.RUnlock()
	select {
	case <-ps.closed:
		return
	default:
	}

	ps.wrapped.Pub(block, toString(block.GetKey()))
}

func (ps *impl) Shutdown() {
	ps.lk.Lock()
	defer ps.lk.Unlock()
	select {
	case <-ps.closed:
		return
	default:
	}
	close(ps.closed)
	ps.wrapped.Shutdown()
}

// Subscribe returns a channel of blocks for the given |keys|. |blockChannel|
// is closed if the |ctx| times out or is cancelled, or after receiving the blocks
// corresponding to |keys|.
func (ps *impl) Subscribe(ctx context.Context, keys ...wl.WantKey) <-chan message.MsgBlock {

	blocksCh := make(chan message.MsgBlock, len(keys))
	valuesCh := make(chan interface{}, len(keys)) // provide our own channel to control buffer, prevent blocking
	if len(keys) == 0 {
		close(blocksCh)
		return blocksCh
	}

	// prevent shutdown
	ps.lk.RLock()
	defer ps.lk.RUnlock()

	select {
	case <-ps.closed:
		close(blocksCh)
		return blocksCh
	default:
	}

	// AddSubOnceEach listens for each key in the list, and closes the channel
	// once all keys have been received
	ps.wrapped.AddSubOnceEach(valuesCh, toStrings(keys)...)
	go func() {
		defer func() {
			close(blocksCh)

			ps.lk.RLock()
			defer ps.lk.RUnlock()
			// Don't touch the pubsub instance if we're
			// already closed.
			select {
			case <-ps.closed:
				return
			default:
			}

			ps.wrapped.Unsub(valuesCh)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ps.closed:
			case val, ok := <-valuesCh:
				if !ok {
					return
				}
				block, ok := val.(message.MsgBlock)
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case blocksCh <- block: // continue
				case <-ps.closed:
				}
			}
		}
	}()

	return blocksCh
}

func toStrings(keys []wl.WantKey) []string {
	strs := make([]string, 0, len(keys))
	for _, key := range keys {
		strs = append(strs, toString(key))
	}
	return strs
}

func toString(key wl.WantKey) string {
	return string(key.Ch) + "-" + key.Cid.KeyString()
}
