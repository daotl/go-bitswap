package sessionmanager

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	bsmsg "github.com/daotl/go-bitswap/message"
	wl "github.com/daotl/go-bitswap/wantlist"
	exchange "github.com/daotl/go-ipfs-exchange-interface"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	delay "github.com/ipfs/go-ipfs-delay"
	"github.com/libp2p/go-libp2p-core/peer"

	bsbpm "github.com/daotl/go-bitswap/internal/blockpresencemanager"
	"github.com/daotl/go-bitswap/internal/notifications"
	bspm "github.com/daotl/go-bitswap/internal/peermanager"
	bssession "github.com/daotl/go-bitswap/internal/session"
	bssim "github.com/daotl/go-bitswap/internal/sessioninterestmanager"
	"github.com/daotl/go-bitswap/internal/testutil"
)

type fakeSession struct {
	ks         []wl.WantKey
	wantBlocks []wl.WantKey
	wantHaves  []wl.WantKey
	id         uint64
	pm         *fakeSesPeerManager
	sm         bssession.SessionManager
	notif      notifications.PubSub
}

func (*fakeSession) GetBlock(context.Context, cid.Cid) (blocks.Block, error) {
	return nil, nil
}
func (*fakeSession) GetBlockFromChannel(context.Context, exchange.Channel, cid.Cid) (
	blocks.Block, error) {
	return nil, nil
}
func (*fakeSession) GetBlocks(context.Context, []cid.Cid) (<-chan blocks.Block, error) {
	return nil, nil
}
func (*fakeSession) GetBlocksFromChannel(context.Context, exchange.Channel, []cid.Cid) (
	<-chan blocks.Block, error) {
	return nil, nil
}
func (fs *fakeSession) ID() uint64 {
	return fs.id
}
func (fs *fakeSession) ReceiveFrom(p peer.ID, ks []wl.WantKey, wantBlocks []wl.WantKey, wantHaves []wl.WantKey) {
	fs.ks = append(fs.ks, ks...)
	fs.wantBlocks = append(fs.wantBlocks, wantBlocks...)
	fs.wantHaves = append(fs.wantHaves, wantHaves...)
}
func (fs *fakeSession) Shutdown() {
	fs.sm.RemoveSession(fs.id)
}

type fakeSesPeerManager struct {
}

func (*fakeSesPeerManager) Peers() []peer.ID          { return nil }
func (*fakeSesPeerManager) PeersDiscovered() bool     { return false }
func (*fakeSesPeerManager) Shutdown()                 {}
func (*fakeSesPeerManager) AddPeer(peer.ID) bool      { return false }
func (*fakeSesPeerManager) RemovePeer(peer.ID) bool   { return false }
func (*fakeSesPeerManager) HasPeers() bool            { return false }
func (*fakeSesPeerManager) ProtectConnection(peer.ID) {}

type fakePeerManager struct {
	lk      sync.Mutex
	cancels []wl.WantKey
}

func (*fakePeerManager) RegisterSession(peer.ID, bspm.Session)                          {}
func (*fakePeerManager) UnregisterSession(uint64)                                       {}
func (*fakePeerManager) SendWants(context.Context, peer.ID, []wl.WantKey, []wl.WantKey) {}
func (*fakePeerManager) BroadcastWantHaves(context.Context, []wl.WantKey)               {}
func (fpm *fakePeerManager) SendCancels(ctx context.Context, cancels []wl.WantKey) {
	fpm.lk.Lock()
	defer fpm.lk.Unlock()
	fpm.cancels = append(fpm.cancels, cancels...)
}
func (fpm *fakePeerManager) cancelled() []wl.WantKey {
	fpm.lk.Lock()
	defer fpm.lk.Unlock()
	return fpm.cancels
}

func sessionFactory(ctx context.Context,
	sm bssession.SessionManager,
	id uint64,
	sprm bssession.SessionPeerManager,
	sim *bssim.SessionInterestManager,
	pm bssession.PeerManager,
	bpm *bsbpm.BlockPresenceManager,
	notif notifications.PubSub,
	provSearchDelay time.Duration,
	rebroadcastDelay delay.D,
	self peer.ID,
	ch exchange.Channel) Session {
	fs := &fakeSession{
		id:    id,
		pm:    sprm.(*fakeSesPeerManager),
		sm:    sm,
		notif: notif,
	}
	go func() {
		<-ctx.Done()
		sm.RemoveSession(fs.id)
	}()
	return fs
}

func peerManagerFactory(ctx context.Context, id uint64) bssession.SessionPeerManager {
	return &fakeSesPeerManager{}
}

func TestReceiveFrom(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	notif := notifications.New()
	defer notif.Shutdown()
	sim := bssim.New()
	bpm := bsbpm.New()
	pm := &fakePeerManager{}
	sm := New(ctx, sessionFactory, sim, peerManagerFactory, bpm, pm, notif, "")

	p := peer.ID(fmt.Sprint(123))
	block := bsmsg.NewMsgBlock(blocks.NewBlock([]byte("block")), "")
	firstSession := sm.NewSession(ctx, time.Second, delay.Fixed(time.Minute), exchange.PublicChannel).(*fakeSession)
	secondSession := sm.NewSession(ctx, time.Second, delay.Fixed(time.Minute), exchange.PublicChannel).(*fakeSession)
	thirdSession := sm.NewSession(ctx, time.Second, delay.Fixed(time.Minute), exchange.PublicChannel).(*fakeSession)

	sim.RecordSessionInterest(firstSession.ID(), []wl.WantKey{block.GetKey()})
	sim.RecordSessionInterest(thirdSession.ID(), []wl.WantKey{block.GetKey()})

	sm.ReceiveFrom(ctx, p, []wl.WantKey{block.GetKey()}, nil, nil)
	if len(firstSession.ks) == 0 ||
		len(secondSession.ks) > 0 ||
		len(thirdSession.ks) == 0 {
		t.Fatal("should have received blocks but didn't")
	}

	sm.ReceiveFrom(ctx, p, nil, []wl.WantKey{block.GetKey()}, nil)
	if len(firstSession.wantBlocks) == 0 ||
		len(secondSession.wantBlocks) > 0 ||
		len(thirdSession.wantBlocks) == 0 {
		t.Fatal("should have received want-blocks but didn't")
	}

	sm.ReceiveFrom(ctx, p, nil, nil, []wl.WantKey{block.GetKey()})
	if len(firstSession.wantHaves) == 0 ||
		len(secondSession.wantHaves) > 0 ||
		len(thirdSession.wantHaves) == 0 {
		t.Fatal("should have received want-haves but didn't")
	}

	if len(pm.cancelled()) != 1 {
		t.Fatal("should have sent cancel for received blocks")
	}
}

func TestReceiveBlocksWhenManagerShutdown(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	notif := notifications.New()
	defer notif.Shutdown()
	sim := bssim.New()
	bpm := bsbpm.New()
	pm := &fakePeerManager{}
	sm := New(ctx, sessionFactory, sim, peerManagerFactory, bpm, pm, notif, "")

	p := peer.ID(fmt.Sprint(123))
	block := bsmsg.NewMsgBlock(blocks.NewBlock([]byte("block")), "")

	firstSession := sm.NewSession(ctx, time.Second, delay.Fixed(time.Minute), exchange.PublicChannel).(*fakeSession)
	secondSession := sm.NewSession(ctx, time.Second, delay.Fixed(time.Minute), exchange.PublicChannel).(*fakeSession)
	thirdSession := sm.NewSession(ctx, time.Second, delay.Fixed(time.Minute), exchange.PublicChannel).(*fakeSession)
	keys := []wl.WantKey{block.GetKey()}

	sim.RecordSessionInterest(firstSession.ID(), keys)
	sim.RecordSessionInterest(secondSession.ID(), keys)
	sim.RecordSessionInterest(thirdSession.ID(), keys)

	sm.Shutdown()

	// wait for sessions to get removed
	time.Sleep(10 * time.Millisecond)

	sm.ReceiveFrom(ctx, p, keys, nil, nil)
	if len(firstSession.ks) > 0 ||
		len(secondSession.ks) > 0 ||
		len(thirdSession.ks) > 0 {
		t.Fatal("received blocks for sessions after manager is shutdown")
	}
}

func TestReceiveBlocksWhenSessionContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	notif := notifications.New()
	defer notif.Shutdown()
	sim := bssim.New()
	bpm := bsbpm.New()
	pm := &fakePeerManager{}
	sm := New(ctx, sessionFactory, sim, peerManagerFactory, bpm, pm, notif, "")

	p := peer.ID(fmt.Sprint(123))
	block := bsmsg.NewMsgBlock(blocks.NewBlock([]byte("block")), "")

	firstSession := sm.NewSession(ctx, time.Second, delay.Fixed(time.Minute), exchange.PublicChannel).(*fakeSession)
	sessionCtx, sessionCancel := context.WithCancel(ctx)
	secondSession := sm.NewSession(sessionCtx, time.Second, delay.Fixed(time.Minute), exchange.PublicChannel).(*fakeSession)
	thirdSession := sm.NewSession(ctx, time.Second, delay.Fixed(time.Minute), exchange.PublicChannel).(*fakeSession)

	keys := []wl.WantKey{block.GetKey()}
	sim.RecordSessionInterest(firstSession.ID(), keys)
	sim.RecordSessionInterest(secondSession.ID(), keys)
	sim.RecordSessionInterest(thirdSession.ID(), keys)

	sessionCancel()

	// wait for sessions to get removed
	time.Sleep(10 * time.Millisecond)

	sm.ReceiveFrom(ctx, p, keys, nil, nil)
	if len(firstSession.ks) == 0 ||
		len(secondSession.ks) > 0 ||
		len(thirdSession.ks) == 0 {
		t.Fatal("received blocks for sessions that are canceled")
	}
}

func TestShutdown(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	notif := notifications.New()
	defer notif.Shutdown()
	sim := bssim.New()
	bpm := bsbpm.New()
	pm := &fakePeerManager{}
	sm := New(ctx, sessionFactory, sim, peerManagerFactory, bpm, pm, notif, "")

	p := peer.ID(fmt.Sprint(123))
	block := bsmsg.NewMsgBlock(blocks.NewBlock([]byte("block")), "")
	keys := []wl.WantKey{block.GetKey()}
	firstSession := sm.NewSession(ctx, time.Second, delay.Fixed(time.Minute), exchange.PublicChannel).(*fakeSession)
	sim.RecordSessionInterest(firstSession.ID(), keys)
	sm.ReceiveFrom(ctx, p, nil, nil, keys)

	if !bpm.HasKey(block.GetKey()) {
		t.Fatal("expected cid to be added to block presence manager")
	}

	sm.Shutdown()

	// wait for cleanup
	time.Sleep(10 * time.Millisecond)

	if bpm.HasKey(block.GetKey()) {
		t.Fatal("expected cid to be removed from block presence manager")
	}
	if !testutil.MatchKeysIgnoreOrder(pm.cancelled(), keys) {
		t.Fatal("expected cancels to be sent")
	}
}
