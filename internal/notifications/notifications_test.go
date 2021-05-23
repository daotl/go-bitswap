package notifications

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/daotl/go-bitswap/message"
	wl "github.com/daotl/go-bitswap/wantlist"
	blocks "github.com/ipfs/go-block-format"
	blocksutil "github.com/ipfs/go-ipfs-blocksutil"
)

func TestDuplicates(t *testing.T) {
	rb1 := blocks.NewBlock([]byte("1"))
	rb2 := blocks.NewBlock([]byte("2"))

	b1 := message.NewMsgBlock(rb1, "")
	b2 := message.NewMsgBlock(rb2, "")
	n := New()
	defer n.Shutdown()
	ch := n.Subscribe(context.Background(), b1.GetKey(), b2.GetKey())

	n.Publish(b1)
	blockRecvd, ok := <-ch
	if !ok {
		t.Fail()
	}
	assertBlocksEqual(t, b1, blockRecvd)

	n.Publish(b1) // ignored duplicate

	n.Publish(b2)
	blockRecvd, ok = <-ch
	if !ok {
		t.Fail()
	}
	assertBlocksEqual(t, b2, blockRecvd)
}

func TestPublishSubscribe(t *testing.T) {
	blockSent := message.NewMsgBlock(blocks.NewBlock([]byte("Greetings from The Interval")), "")

	n := New()
	defer n.Shutdown()
	ch := n.Subscribe(context.Background(), blockSent.GetKey())

	n.Publish(blockSent)
	blockRecvd, ok := <-ch
	if !ok {
		t.Fail()
	}

	assertBlocksEqual(t, blockRecvd, blockSent)

}

func TestSubscribeMany(t *testing.T) {
	e1 := message.NewMsgBlock(blocks.NewBlock([]byte("1")), "")
	e2 := message.NewMsgBlock(blocks.NewBlock([]byte("2")), "")

	n := New()
	defer n.Shutdown()
	ch := n.Subscribe(context.Background(), e1.GetKey(), e2.GetKey())

	n.Publish(e1)
	r1, ok := <-ch
	if !ok {
		t.Fatal("didn't receive first expected block")
	}
	assertBlocksEqual(t, e1, r1)

	n.Publish(e2)
	r2, ok := <-ch
	if !ok {
		t.Fatal("didn't receive second expected block")
	}
	assertBlocksEqual(t, e2, r2)
}

// TestDuplicateSubscribe tests a scenario where a given block
// would be requested twice at the same time.
func TestDuplicateSubscribe(t *testing.T) {
	e1 := message.NewMsgBlock(blocks.NewBlock([]byte("1")), "")

	n := New()
	defer n.Shutdown()
	ch1 := n.Subscribe(context.Background(), e1.GetKey())
	ch2 := n.Subscribe(context.Background(), e1.GetKey())

	n.Publish(e1)
	r1, ok := <-ch1
	if !ok {
		t.Fatal("didn't receive first expected block")
	}
	assertBlocksEqual(t, e1, r1)

	r2, ok := <-ch2
	if !ok {
		t.Fatal("didn't receive second expected block")
	}
	assertBlocksEqual(t, e1, r2)
}

func TestShutdownBeforeUnsubscribe(t *testing.T) {
	e1 := message.NewMsgBlock(blocks.NewBlock([]byte("1")), "")

	n := New()
	ctx, cancel := context.WithCancel(context.Background())
	ch := n.Subscribe(ctx, e1.GetKey()) // no keys provided
	n.Shutdown()
	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("channel should have been closed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("channel should have been closed")
	}
}

func TestSubscribeIsANoopWhenCalledWithNoKeys(t *testing.T) {
	n := New()
	defer n.Shutdown()
	ch := n.Subscribe(context.Background()) // no keys provided
	if _, ok := <-ch; ok {
		t.Fatal("should be closed if no keys provided")
	}
}

func TestCarryOnWhenDeadlineExpires(t *testing.T) {

	impossibleDeadline := time.Nanosecond
	fastExpiringCtx, cancel := context.WithTimeout(context.Background(), impossibleDeadline)
	defer cancel()

	n := New()
	defer n.Shutdown()
	block := message.NewMsgBlock(blocks.NewBlock([]byte("A Missed Connection")), "")
	blockChannel := n.Subscribe(fastExpiringCtx, block.GetKey())

	assertBlockChannelNil(t, blockChannel)
}

func TestDoesNotDeadLockIfContextCancelledBeforePublish(t *testing.T) {

	g := blocksutil.NewBlockGenerator()
	ctx, cancel := context.WithCancel(context.Background())
	n := New()
	defer n.Shutdown()

	t.Log("generate a large number of blocks. exceed default buffer")
	bs := g.Blocks(1000)
	ks := func() []wl.WantKey {
		var keys []wl.WantKey
		for _, b := range bs {
			keys = append(keys, wl.NewWantKey(b.Cid(), ""))
		}
		return keys
	}()

	_ = n.Subscribe(ctx, ks...) // ignore received channel

	t.Log("cancel context before any blocks published")
	cancel()
	for _, b := range bs {
		n.Publish(message.NewMsgBlock(b, ""))
	}

	t.Log("publishing the large number of blocks to the ignored channel must not deadlock")
}

func assertBlockChannelNil(t *testing.T, blockChannel <-chan message.MsgBlock) {
	_, ok := <-blockChannel
	if ok {
		t.Fail()
	}
}

func assertBlocksEqual(t *testing.T, a, b message.MsgBlock) {
	if !bytes.Equal(a.RawData(), b.RawData()) {
		t.Fatal("blocks aren't equal")
	}
	if a.GetKey() != b.GetKey() {
		t.Fatal("block keys aren't equal")
	}
}
