package message

import (
	"bytes"
	"testing"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	blocksutil "github.com/ipfs/go-ipfs-blocksutil"
	u "github.com/ipfs/go-ipfs-util"

	pb "github.com/daotl/go-bitswap/message/pb"
	"github.com/daotl/go-bitswap/wantlist"
)

func mkFakeKey(s string) wantlist.WantKey {
	return wantlist.NewWantKey(cid.NewCidV0(u.Hash([]byte(s))), "")
}

func mkFakeMsgBlock(bs []byte) MsgBlock {
	return NewMsgBlock(blocks.NewBlock(bs), "")
}

func TestAppendWanted(t *testing.T) {
	str := mkFakeKey("foo")
	m := New(true)
	m.AddEntry(str, 1, pb.Message_Wantlist_Block, true)

	if !wantlistContains(&m.ToProtoV0().Wantlist, str) {
		t.Fail()
	}
}

func TestNewMessageFromProto(t *testing.T) {
	str := mkFakeKey("a_key")
	protoMessage := new(pb.Message)
	protoMessage.Wantlist.Entries = []pb.Message_Wantlist_Entry{
		{Block: pb.Cid{Cid: str.Cid}},
	}
	if !wantlistContains(&protoMessage.Wantlist, str) {
		t.Fail()
	}
	m, err := newMessageFromProto(*protoMessage)
	if err != nil {
		t.Fatal(err)
	}

	if !wantlistContains(&m.ToProtoV0().Wantlist, str) {
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
		m.AddBlock(NewMsgBlock(block, ""))
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
	keystrs := []wantlist.WantKey{mkFakeKey("foo"), mkFakeKey("bar"), mkFakeKey("baz"), mkFakeKey("bat")}
	m := New(true)
	for _, s := range keystrs {
		m.AddEntry(s, 1, pb.Message_Wantlist_Block, true)
	}
	exported := m.Wantlist()

	for _, k := range exported {
		present := false
		for _, s := range keystrs {

			if s.Equals(k.Key) {
				present = true
			}
		}
		if !present {
			t.Logf("%v isn't in original list", k.Key)
			t.Fail()
		}
	}
}

func TestCopyProtoByValue(t *testing.T) {
	str := mkFakeKey("foo")
	m := New(true)
	protoBeforeAppend := m.ToProtoV0()
	m.AddEntry(str, 1, pb.Message_Wantlist_Block, true)
	if wantlistContains(&protoBeforeAppend.Wantlist, str) {
		t.Fail()
	}
}

func TestToNetFromNetPreservesWantList(t *testing.T) {
	original := New(true)
	original.AddEntry(mkFakeKey("M"), 1, pb.Message_Wantlist_Block, true)
	original.AddEntry(mkFakeKey("B"), 1, pb.Message_Wantlist_Block, true)
	original.AddEntry(mkFakeKey("D"), 1, pb.Message_Wantlist_Block, true)
	original.AddEntry(mkFakeKey("T"), 1, pb.Message_Wantlist_Block, true)
	original.AddEntry(mkFakeKey("F"), 1, pb.Message_Wantlist_Block, true)

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

	keys := make(map[wantlist.WantKey]bool)
	for _, k := range copied.Wantlist() {
		keys[k.Key] = true
	}

	for _, k := range original.Wantlist() {
		if _, ok := keys[k.Key]; !ok {
			t.Fatalf("Key Missing: \"%v\"", k)
		}
	}
}

func TestToAndFromNetMessage(t *testing.T) {

	original := New(true)
	original.AddBlock(mkFakeMsgBlock([]byte("W")))
	original.AddBlock(mkFakeMsgBlock([]byte("E")))
	original.AddBlock(mkFakeMsgBlock([]byte("F")))
	original.AddBlock(mkFakeMsgBlock([]byte("M")))

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

func wantlistContains(wantlist *pb.Message_Wantlist, c wantlist.WantKey) bool {
	for _, e := range wantlist.GetEntries() {
		if e.Block.Cid.Defined() && c.Cid.Equals(e.Block.Cid) && string(c.Ch) == e.Channel {
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
	b := mkFakeMsgBlock([]byte("foo"))
	msg := New(true)

	msg.AddEntry(b.GetKey(), 1, pb.Message_Wantlist_Block, true)
	msg.AddEntry(b.GetKey(), 1, pb.Message_Wantlist_Block, true)
	if len(msg.Wantlist()) != 1 {
		t.Fatal("Duplicate in BitSwapMessage")
	}

	msg.AddBlock(b)
	msg.AddBlock(b)
	if len(msg.Blocks()) != 1 {
		t.Fatal("Duplicate in BitSwapMessage")
	}

	b2 := mkFakeMsgBlock([]byte("bar"))
	msg.AddBlockPresence(b2.GetKey(), pb.Message_Have)
	msg.AddBlockPresence(b2.GetKey(), pb.Message_Have)
	if len(msg.Haves()) != 1 {
		t.Fatal("Duplicate in BitSwapMessage")
	}
}

func TestBlockPresences(t *testing.T) {
	b1 := mkFakeMsgBlock([]byte("foo"))
	b2 := mkFakeMsgBlock([]byte("bar"))
	msg := New(true)

	msg.AddBlockPresence(b1.GetKey(), pb.Message_Have)
	msg.AddBlockPresence(b2.GetKey(), pb.Message_DontHave)
	if len(msg.Haves()) != 1 || !msg.Haves()[0].Equals(b1.GetKey()) {
		t.Fatal("Expected HAVE")
	}
	if len(msg.DontHaves()) != 1 || !msg.DontHaves()[0].Equals(b2.GetKey()) {
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

	msg.AddBlockPresence(b1.GetKey(), pb.Message_Have)
	if len(msg.Haves()) != 0 {
		t.Fatal("Expected HAVE not to overwrite block")
	}

	msg.AddBlockPresence(b2.GetKey(), pb.Message_DontHave)
	if len(msg.DontHaves()) != 0 {
		t.Fatal("Expected DONT_HAVE not to overwrite block")
	}
}

func TestAddWantlistEntry(t *testing.T) {
	b := mkFakeMsgBlock([]byte("foo"))
	msg := New(true)

	msg.AddEntry(b.GetKey(), 1, pb.Message_Wantlist_Have, false)
	msg.AddEntry(b.GetKey(), 2, pb.Message_Wantlist_Block, true)
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

	msg.AddEntry(b.GetKey(), 2, pb.Message_Wantlist_Block, true)
	e = msg.Wantlist()[0]
	if e.Priority != 2 {
		t.Fatal("priority should be overridden if wants are of same type")
	}

	msg.AddEntry(b.GetKey(), 3, pb.Message_Wantlist_Have, false)
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

	msg.Cancel(b.GetKey())
	e = msg.Wantlist()[0]
	if !e.Cancel {
		t.Fatal("cancel should override want")
	}

	msg.AddEntry(b.GetKey(), 10, pb.Message_Wantlist_Block, true)
	if !e.Cancel {
		t.Fatal("want should not override cancel")
	}
}

func TestEntrySize(t *testing.T) {
	blockGenerator := blocksutil.NewBlockGenerator()
	c := blockGenerator.Next().Cid()
	e := Entry{
		Entry: wantlist.Entry{
			Key:      wantlist.NewWantKey(c, ""),
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
