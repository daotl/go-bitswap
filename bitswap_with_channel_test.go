package bitswap_test

import (
	"context"
	"testing"
	"time"

	ac "github.com/daotl/go-bitswap/accesscontrol"
	testinstance "github.com/daotl/go-bitswap/testinstance"
	"github.com/ipfs/go-cid"
	blocksutil "github.com/ipfs/go-ipfs-blocksutil"
	"github.com/libp2p/go-libp2p-core/peer"
)

const NonPublicChannel = "another"

func TestWithChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vnet := getVirtualNetwork()
	ig := testinstance.NewTestInstanceGenerator(vnet, nil, nil)
	defer ig.Close()
	bgen := blocksutil.NewBlockGenerator()

	blks := bgen.Blocks(2)
	inst := ig.Instances(2)

	a := inst[0]
	b := inst[1]

	// Add  blocks to Peer B
	if err := b.Blockstore().PutMany(blks); err != nil {
		t.Fatal(err)
	}
	cid0, cid1 := blks[0].Cid(), blks[1].Cid()

	// channel only contains block 0
	ac.GlobalFilter = ac.FixedChannel(NonPublicChannel, []peer.ID{a.Peer, b.Peer}, []cid.Cid{cid0})
	defer ac.ResetFilter()
	// Create a session on Peer A
	sesa := a.Exchange.NewSession(ctx)

	// Get the block 0, should success
	blkout, err := sesa.GetBlockFromChannel(ctx, NonPublicChannel, cid0)
	if err != nil {
		t.Fatal(err)
	}

	if !blkout.Cid().Equals(cid0) {
		t.Fatal("got wrong block")
	}

	blkCh, err := sesa.GetBlocksFromChannel(ctx, NonPublicChannel, []cid.Cid{cid0})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case bk := <-blkCh:
		if !bk.Cid().Equals(cid0) {
			t.Fatal("got wrong block")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected a block")
	}

	blkout1, err := a.Exchange.GetBlockFromChannel(ctx, NonPublicChannel, cid0)
	if err != nil {
		t.Fatal(err)
	}

	if !blkout1.Cid().Equals(cid0) {
		t.Fatal("got wrong block")
	}

	blkCh1, err := a.Exchange.GetBlocksFromChannel(ctx, NonPublicChannel, []cid.Cid{cid0})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case bk := <-blkCh1:
		if !bk.Cid().Equals(cid0) {
			t.Fatal("got wrong block")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected a block")
	}

	// Get the block 1, should fail as it is not included in this channel
	_, err = sesa.GetBlockFromChannel(ctx, NonPublicChannel, cid1)
	if err == nil {
		t.Fatal("expected blocked")
	}
	_, err = sesa.GetBlocksFromChannel(ctx, NonPublicChannel, []cid.Cid{cid1})
	if err == nil {
		t.Fatal("expected blocked")
	}

	_, err = a.Exchange.GetBlockFromChannel(ctx, NonPublicChannel, cid1)
	if err == nil {
		t.Fatal("expected blocked")
	}
	_, err = a.Exchange.GetBlocksFromChannel(ctx, NonPublicChannel, []cid.Cid{cid1})
	if err == nil {
		t.Fatal("expected blocked")
	}
}

func TestBroadcastFiltered(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vnet := getVirtualNetwork()
	ig := testinstance.NewTestInstanceGenerator(vnet, nil, nil)
	defer ig.Close()
	bgen := blocksutil.NewBlockGenerator()

	blks := bgen.Blocks(1)
	inst := ig.Instances(3)

	a := inst[0]
	b := inst[1]
	c := inst[2]

	// Add  blocks to Peer B
	if err := b.Blockstore().PutMany(blks); err != nil {
		t.Fatal(err)
	}

	// Create a session on Peer A
	sesa := a.Exchange.NewSession(ctx)

	// channel only contains block 0
	ac.GlobalFilter = ac.FixedChannel("", []peer.ID{a.Peer, b.Peer}, []cid.Cid{blks[0].Cid()})
	defer ac.ResetFilter()
	// Get the block 0, should broadcast want-have to b, not c
	blkout, err := sesa.GetBlock(ctx, blks[0].Cid())
	if err != nil {
		t.Fatal(err)
	}

	if !blkout.Cid().Equals(blks[0].Cid()) {
		t.Fatal("got wrong block")
	}

	//stata, _ := a.Exchange.Stat()
	//statb, _ := b.Exchange.Stat()
	statc, _ := c.Exchange.Stat()
	//println(stata.MessagesReceived)
	//println(statb.MessagesReceived)
	if statc.MessagesReceived != 0 {
		t.Fatal("expected no msg received")
	}

}
