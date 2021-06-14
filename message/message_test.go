package message

import (
	"bytes"
	"testing"

	"github.com/daotl/go-ipld-channel/pair"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	blocksutil "github.com/ipfs/go-ipfs-blocksutil"
	u "github.com/ipfs/go-ipfs-util"

	pb "github.com/daotl/go-bitswap/message/pb"
	"github.com/daotl/go-bitswap/wantlist"
)

func mkFakeCid(s string) cid.Cid {
	return cid.NewCidV0(u.Hash([]byte(s)))
}

func mkFakePair(s string) pair.CidChannelPair {
	return pair.PublicCidPair(mkFakeCid(s))
}

func TestAppendWanted(t *testing.T) {
	p := mkFakePair("foo")
	m := New(true)
	m.AddEntry(p, 1, pb.Message_Wantlist_Block, true)

	if !wantlistContains(&m.ToProtoV0().Wantlist, p) {
		t.Fail()
	}
}

func TestNewMessageFromProto(t *testing.T) {
	p := mkFakePair("a_key")
	protoMessage := new(pb.Message)
	protoMessage.Wantlist.Entries = []pb.Message_Wantlist_Entry{
		{Block: pb.Cid{Cid: p.Cid}},
	}
	if !wantlistContains(&protoMessage.Wantlist, p) {
		t.Fail()
	}
	m, err := newMessageFromProto(*protoMessage)
	if err != nil {
		t.Fatal(err)
	}

	if !wantlistContains(&m.ToProtoV0().Wantlist, p) {
		t.Fail()
	}
}

func TestAppendBlock(t *testing.T) {

	strs := make([]string, 2)
	strs = append(strs, "Celeritas")
	strs = append(strs, "Incendia")

	m := New(true)
	for _, str := range strs {
		block := blocks.NewBlock([]byte(str))
		m.AddBlock(block)
	}

	// assert strings are in proto message
	for _, blockbytes := range m.ToProtoV0().GetBlocks() {
		s := bytes.NewBuffer(blockbytes).String()
		if !contains(strs, s) {
			t.Fail()
		}
	}
}

func TestWantlist(t *testing.T) {
	keypairs := []pair.CidChannelPair{mkFakePair("foo"), mkFakePair("bar"), mkFakePair("baz"),
		mkFakePair("bat")}
	m := New(true)
	for _, p := range keypairs {
		m.AddEntry(p, 1, pb.Message_Wantlist_Block, true)
	}
	exported := m.Wantlist()

	for _, k := range exported {
		present := false
		for _, p := range keypairs {

			if p.Equals(k.Pair) {
				present = true
			}
		}
		if !present {
			t.Logf("%v isn't in original list", k.Pair)
			t.Fail()
		}
	}
}

func TestCopyProtoByValue(t *testing.T) {
	p := mkFakePair("foo")
	m := New(true)
	protoBeforeAppend := m.ToProtoV0()
	m.AddEntry(p, 1, pb.Message_Wantlist_Block, true)
	if wantlistContains(&protoBeforeAppend.Wantlist, p) {
		t.Fail()
	}
}

func TestToNetFromNetPreservesWantList(t *testing.T) {
	original := New(true)
	original.AddEntry(mkFakePair("M"), 1, pb.Message_Wantlist_Block, true)
	original.AddEntry(mkFakePair("B"), 1, pb.Message_Wantlist_Block, true)
	original.AddEntry(mkFakePair("D"), 1, pb.Message_Wantlist_Block, true)
	original.AddEntry(mkFakePair("T"), 1, pb.Message_Wantlist_Block, true)
	original.AddEntry(mkFakePair("F"), 1, pb.Message_Wantlist_Block, true)

	buf := new(bytes.Buffer)
	if err := original.ToNetV1(buf); err != nil {
		t.Fatal(err)
	}

	copied, err := FromNet(buf)
	if err != nil {
		t.Fatal(err)
	}

	if !copied.Full() {
		t.Fatal("fullness attribute got dropped on marshal")
	}

	pairs := make(map[pair.CidChannelPair]bool)
	for _, k := range copied.Wantlist() {
		pairs[k.Pair] = true
	}

	for _, k := range original.Wantlist() {
		if _, ok := pairs[k.Pair]; !ok {
			t.Fatalf("Pair Missing: \"%v\"", k)
		}
	}
}

func TestToAndFromNetMessage(t *testing.T) {

	original := New(true)
	original.AddBlock(blocks.NewBlock([]byte("W")))
	original.AddBlock(blocks.NewBlock([]byte("E")))
	original.AddBlock(blocks.NewBlock([]byte("F")))
	original.AddBlock(blocks.NewBlock([]byte("M")))

	buf := new(bytes.Buffer)
	if err := original.ToNetV1(buf); err != nil {
		t.Fatal(err)
	}

	m2, err := FromNet(buf)
	if err != nil {
		t.Fatal(err)
	}

	keys := make(map[cid.Cid]bool)
	for _, b := range m2.Blocks() {
		keys[b.Cid()] = true
	}

	for _, b := range original.Blocks() {
		if _, ok := keys[b.Cid()]; !ok {
			t.Fail()
		}
	}
}

func wantlistContains(wantlist *pb.Message_Wantlist, p pair.CidChannelPair) bool {
	for _, e := range wantlist.GetEntries() {
		if e.Block.Cid.Defined() && p.Cid.Equals(e.Block.Cid) && string(p.Channel) == e.Channel {
			return true
		}
	}
	return false
}

func contains(strs []string, x string) bool {
	for _, s := range strs {
		if s == x {
			return true
		}
	}
	return false
}

func TestDuplicates(t *testing.T) {
	b := blocks.NewBlock([]byte("foo"))
	p := pair.PublicCidPair(b.Cid())
	msg := New(true)

	msg.AddEntry(p, 1, pb.Message_Wantlist_Block, true)
	msg.AddEntry(p, 1, pb.Message_Wantlist_Block, true)
	if len(msg.Wantlist()) != 1 {
		t.Fatal("Duplicate in BitSwapMessage")
	}

	msg.AddBlock(b)
	msg.AddBlock(b)
	if len(msg.Blocks()) != 1 {
		t.Fatal("Duplicate in BitSwapMessage")
	}

	b2 := blocks.NewBlock([]byte("bar"))
	p2 := pair.PublicCidPair(b2.Cid())
	msg.AddBlockPresence(p2, pb.Message_Have)
	msg.AddBlockPresence(p2, pb.Message_Have)
	if len(msg.Haves()) != 1 {
		t.Fatal("Duplicate in BitSwapMessage")
	}
}

func TestBlockPresences(t *testing.T) {
	b1 := blocks.NewBlock([]byte("foo"))
	p1 := pair.PublicCidPair(b1.Cid())
	b2 := blocks.NewBlock([]byte("bar"))
	p2 := pair.PublicCidPair(b2.Cid())
	msg := New(true)

	msg.AddBlockPresence(p1, pb.Message_Have)
	msg.AddBlockPresence(p2, pb.Message_DontHave)
	if len(msg.Haves()) != 1 || !msg.Haves()[0].Equals(p1) {
		t.Fatal("Expected HAVE")
	}
	if len(msg.DontHaves()) != 1 || !msg.DontHaves()[0].Equals(p2) {
		t.Fatal("Expected HAVE")
	}

	msg.AddBlock(b1)
	if len(msg.Haves()) != 0 {
		t.Fatal("Expected block to overwrite HAVE")
	}

	msg.AddBlock(b2)
	if len(msg.DontHaves()) != 0 {
		t.Fatal("Expected block to overwrite DONT_HAVE")
	}

	msg.AddBlockPresence(p1, pb.Message_Have)
	if len(msg.Haves()) != 0 {
		t.Fatal("Expected HAVE not to overwrite block")
	}

	msg.AddBlockPresence(p2, pb.Message_DontHave)
	if len(msg.DontHaves()) != 0 {
		t.Fatal("Expected DONT_HAVE not to overwrite block")
	}
}

func TestAddWantlistEntry(t *testing.T) {
	b := blocks.NewBlock([]byte("foo"))
	p := pair.PublicCidPair(b.Cid())
	msg := New(true)

	msg.AddEntry(p, 1, pb.Message_Wantlist_Have, false)
	msg.AddEntry(p, 2, pb.Message_Wantlist_Block, true)
	entries := msg.Wantlist()
	if len(entries) != 1 {
		t.Fatal("Duplicate in BitSwapMessage")
	}
	e := entries[0]
	if e.WantType != pb.Message_Wantlist_Block {
		t.Fatal("want-block should override want-have")
	}
	if e.SendDontHave != true {
		t.Fatal("true SendDontHave should override false SendDontHave")
	}
	if e.Priority != 1 {
		t.Fatal("priority should only be overridden if wants are of same type")
	}

	msg.AddEntry(p, 2, pb.Message_Wantlist_Block, true)
	e = msg.Wantlist()[0]
	if e.Priority != 2 {
		t.Fatal("priority should be overridden if wants are of same type")
	}

	msg.AddEntry(p, 3, pb.Message_Wantlist_Have, false)
	e = msg.Wantlist()[0]
	if e.WantType != pb.Message_Wantlist_Block {
		t.Fatal("want-have should not override want-block")
	}
	if e.SendDontHave != true {
		t.Fatal("false SendDontHave should not override true SendDontHave")
	}
	if e.Priority != 2 {
		t.Fatal("priority should only be overridden if wants are of same type")
	}

	msg.Cancel(p)
	e = msg.Wantlist()[0]
	if !e.Cancel {
		t.Fatal("cancel should override want")
	}

	msg.AddEntry(p, 10, pb.Message_Wantlist_Block, true)
	if !e.Cancel {
		t.Fatal("want should not override cancel")
	}
}

func TestEntrySize(t *testing.T) {
	blockGenerator := blocksutil.NewBlockGenerator()
	c := blockGenerator.Next().Cid()
	e := Entry{
		Entry: wantlist.Entry{
			Pair:     pair.PublicCidPair(c),
			Priority: 10,
			WantType: pb.Message_Wantlist_Have,
		},
		SendDontHave: true,
		Cancel:       false,
	}
	epb := e.ToPB()
	if e.Size() != epb.Size() {
		t.Fatal("entry size calculation incorrect", e.Size(), epb.Size())
	}
}
