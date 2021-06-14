package message

import (
	"encoding/binary"
	"errors"
	"io"

	channel "github.com/daotl/go-ipld-channel"
	"github.com/daotl/go-ipld-channel/pair"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	u "github.com/ipfs/go-ipfs-util"
	pool "github.com/libp2p/go-buffer-pool"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-msgio"

	pb "github.com/daotl/go-bitswap/message/pb"
	"github.com/daotl/go-bitswap/wantlist"
)

// BitSwapMessage is the basic interface for interacting building, encoding,
// and decoding messages sent on the BitSwap protocol.
type BitSwapMessage interface {
	// Wantlist returns a slice of unique keys that represent data wanted by
	// the sender.
	Wantlist() []Entry

	// Blocks returns a slice of unique blocks.
	Blocks() []blocks.Block
	// BlockPresences returns the list of HAVE / DONT_HAVE in the message
	BlockPresences() []BlockPresence
	// Haves returns the CidChannelPairs for each HAVE
	Haves() []pair.CidChannelPair
	// DontHaves returns the BlockPresences for each DONT_HAVE
	DontHaves() []pair.CidChannelPair
	// PendingBytes returns the number of outstanding bytes of data that the
	// engine has yet to send to the client (because they didn't fit in this
	// message)
	PendingBytes() int32

	// AddEntry adds an entry to the Wantlist.
	AddEntry(key pair.CidChannelPair, priority int32, wantType pb.Message_Wantlist_WantType,
		sendDontHave bool) int

	// Cancel adds a CANCEL for the given CidChannelPair to the message
	// Returns the size of the CANCEL entry in the protobuf
	Cancel(key pair.CidChannelPair) int

	// Remove removes any entries for the given CidChannelPair. Useful when the want
	// status for the CidChannelPair changes when preparing a message.
	Remove(key pair.CidChannelPair)

	// Empty indicates whether the message has any information
	Empty() bool
	// Size returns the size of the message in bytes
	Size() int

	// A full wantlist is an authoritative copy, a 'non-full' wantlist is a patch-set
	Full() bool

	// AddBlock adds a block to the message
	AddBlock(blocks.Block)
	// AddBlockPresence adds a HAVE / DONT_HAVE for the given CidChannelPair to the message
	AddBlockPresence(pair.CidChannelPair, pb.Message_BlockPresenceType)
	// AddHave adds a HAVE for the given CidChannelPair to the message
	AddHave(pair.CidChannelPair)
	// AddDontHave adds a DONT_HAVE for the given CidChannelPair to the message
	AddDontHave(pair.CidChannelPair)
	// SetPendingBytes sets the number of bytes of data that are yet to be sent
	// to the client (because they didn't fit in this message)
	SetPendingBytes(int32)
	Exportable

	Loggable() map[string]interface{}

	// Reset the values in the message back to defaults, so it can be reused
	Reset(bool)

	// Clone the message fields
	Clone() BitSwapMessage
}

// Exportable is an interface for structures than can be
// encoded in a bitswap protobuf.
type Exportable interface {
	// Note that older Bitswap versions use a different wire format, so we need
	// to convert the message to the appropriate format depending on which
	// version of the protocol the remote peer supports.
	ToProtoV0() *pb.Message
	ToProtoV1() *pb.Message
	ToNetV0(w io.Writer) error
	ToNetV1(w io.Writer) error
}

// BlockPresence represents a HAVE / DONT_HAVE for a given CidChannelPair
type BlockPresence struct {
	Pair pair.CidChannelPair
	Type pb.Message_BlockPresenceType
}

// Entry is a wantlist entry in a Bitswap message, with flags indicating
// - whether message is a cancel
// - whether requester wants a DONT_HAVE message
// - whether requester wants a HAVE message (instead of the block)
type Entry struct {
	wantlist.Entry
	Cancel       bool
	SendDontHave bool
}

// Get the size of the entry on the wire
func (e *Entry) Size() int {
	epb := e.ToPB()
	return epb.Size()
}

// Get the entry in protobuf form
func (e *Entry) ToPB() pb.Message_Wantlist_Entry {
	return pb.Message_Wantlist_Entry{
		Block:        pb.Cid{Cid: e.Pair.Cid},
		Priority:     int32(e.Priority),
		Cancel:       e.Cancel,
		WantType:     e.WantType,
		SendDontHave: e.SendDontHave,
		Channel:      e.Pair.Channel.String(),
	}
}

var MaxEntrySize = maxEntrySize()

func maxEntrySize() int {
	var maxInt32 int32 = (1 << 31) - 1

	c := cid.NewCidV0(u.Hash([]byte("cid")))
	e := Entry{
		Entry: wantlist.Entry{
			// Suppose we use CIDs as IPLD channel identifiers
			Pair:     pair.CidChannelPair{c, channel.Channel(c.String())},
			Priority: maxInt32,
			WantType: pb.Message_Wantlist_Have,
		},
		SendDontHave: true, // true takes up more space than false
		Cancel:       true,
	}
	return e.Size()
}

var _ BitSwapMessage = (*impl)(nil)

type impl struct {
	full           bool
	wantlist       map[pair.CidChannelPair]*Entry
	blocks         map[cid.Cid]blocks.Block
	blockPresences map[cid.Cid]blockPresenceInfos
	pendingBytes   int32
}

type blockPresenceInfos = map[pb.Message_BlockPresenceType]channelMap

type channelMap map[channel.Channel]struct{}

func (m channelMap) toChannelStrings() []string {
	strs := make([]string, 0, len(m))
	for chn := range m {
		strs = append(strs, chn.String())
	}
	return strs
}

// New returns a new, empty bitswap message
func New(full bool) BitSwapMessage {
	return newMsg(full)
}

func newMsg(full bool) *impl {
	return &impl{
		full:           full,
		wantlist:       make(map[pair.CidChannelPair]*Entry),
		blocks:         make(map[cid.Cid]blocks.Block),
		blockPresences: make(map[cid.Cid]blockPresenceInfos),
	}
}

// Clone the message fields
func (m *impl) Clone() BitSwapMessage {
	msg := newMsg(m.full)
	for k := range m.wantlist {
		msg.wantlist[k] = m.wantlist[k]
	}
	for k := range m.blocks {
		msg.blocks[k] = m.blocks[k]
	}
	for k := range m.blockPresences {
		msg.blockPresences[k] = m.blockPresences[k]
	}
	msg.pendingBytes = m.pendingBytes
	return msg
}

// Reset the values in the message back to defaults, so it can be reused
func (m *impl) Reset(full bool) {
	m.full = full
	for k := range m.wantlist {
		delete(m.wantlist, k)
	}
	for k := range m.blocks {
		delete(m.blocks, k)
	}
	for k := range m.blockPresences {
		delete(m.blockPresences, k)
	}
	m.pendingBytes = 0
}

var errCidMissing = errors.New("missing cid")

func newMessageFromProto(pbm pb.Message) (BitSwapMessage, error) {
	m := newMsg(pbm.Wantlist.Full)
	for _, e := range pbm.Wantlist.Entries {
		if !e.Block.Cid.Defined() {
			return nil, errCidMissing
		}
		m.addEntry(pair.CidChannelPair{e.Block.Cid, channel.Channel(e.Channel)},
			e.Priority,
			e.Cancel,
			e.WantType,
			e.SendDontHave)
	}

	// deprecated
	for _, d := range pbm.Blocks {
		// CIDv0, sha256, protobuf only
		b := blocks.NewBlock(d)
		m.AddBlock(b)
	}
	//

	for _, b := range pbm.GetPayload() {
		pref, err := cid.PrefixFromBytes(b.GetPrefix())
		if err != nil {
			return nil, err
		}

		c, err := pref.Sum(b.GetData())
		if err != nil {
			return nil, err
		}

		blk, err := blocks.NewBlockWithCid(b.GetData(), c)
		if err != nil {
			return nil, err
		}

		m.AddBlock(blk)
	}

	for _, bi := range pbm.GetBlockPresences() {
		if !bi.Cid.Cid.Defined() {
			return nil, errCidMissing
		}
		m.addBlockPresences(bi.Cid.Cid, bi.Type, channel.StringsToChannels(bi.Channels)...)
	}

	m.pendingBytes = pbm.PendingBytes

	return m, nil
}

func (m *impl) Full() bool {
	return m.full
}

func (m *impl) Empty() bool {
	return len(m.blocks) == 0 && len(m.wantlist) == 0 && len(m.blockPresences) == 0
}

func (m *impl) Wantlist() []Entry {
	out := make([]Entry, 0, len(m.wantlist))
	for _, e := range m.wantlist {
		out = append(out, *e)
	}
	return out
}

func (m *impl) Blocks() []blocks.Block {
	bs := make([]blocks.Block, 0, len(m.blocks))
	for _, block := range m.blocks {
		bs = append(bs, block)
	}
	return bs
}

func (m *impl) BlockPresences() []BlockPresence {
	bps := make([]BlockPresence, 0, len(m.blockPresences))
	for c, infos := range m.blockPresences {
		for t, chnmap := range infos {
			for chn := range chnmap {
				bps = append(bps, BlockPresence{pair.CidChannelPair{c, chn}, t})
			}
		}
	}
	return bps
}

func (m *impl) Haves() []pair.CidChannelPair {
	return m.getBlockPresenceByType(pb.Message_Have)
}

func (m *impl) DontHaves() []pair.CidChannelPair {
	return m.getBlockPresenceByType(pb.Message_DontHave)
}

func (m *impl) getBlockPresenceByType(t pb.Message_BlockPresenceType) []pair.CidChannelPair {
	pairs := []pair.CidChannelPair{}
	for c, infos := range m.blockPresences {
		for chn := range infos[t] {
			pairs = append(pairs, pair.CidChannelPair{c, chn})
		}
	}
	return pairs
}

func (m *impl) PendingBytes() int32 {
	return m.pendingBytes
}

func (m *impl) SetPendingBytes(pendingBytes int32) {
	m.pendingBytes = pendingBytes
}

func (m *impl) Remove(p pair.CidChannelPair) {
	delete(m.wantlist, p)
}

func (m *impl) Cancel(p pair.CidChannelPair) int {
	return m.addEntry(p, 0, true, pb.Message_Wantlist_Block, false)
}

func (m *impl) AddEntry(p pair.CidChannelPair, priority int32,
	wantType pb.Message_Wantlist_WantType, sendDontHave bool) int {
	return m.addEntry(p, priority, false, wantType, sendDontHave)
}

func (m *impl) addEntry(p pair.CidChannelPair, priority int32, cancel bool,
	wantType pb.Message_Wantlist_WantType, sendDontHave bool) int {
	e, exists := m.wantlist[p]
	if exists {
		// Only change priority if want is of the same type
		if e.WantType == wantType {
			e.Priority = priority
		}
		// Only change from "dont cancel" to "do cancel"
		if cancel {
			e.Cancel = cancel
		}
		// Only change from "dont send" to "do send" DONT_HAVE
		if sendDontHave {
			e.SendDontHave = sendDontHave
		}
		// want-block overrides existing want-have
		if wantType == pb.Message_Wantlist_Block && e.WantType == pb.Message_Wantlist_Have {
			e.WantType = wantType
		}
		m.wantlist[p] = e
		return 0
	}

	e = &Entry{
		Entry: wantlist.Entry{
			Pair:     p,
			Priority: priority,
			WantType: wantType,
		},
		SendDontHave: sendDontHave,
		Cancel:       cancel,
	}
	m.wantlist[p] = e

	return e.Size()
}

func (m *impl) AddBlock(b blocks.Block) {
	delete(m.blockPresences, b.Cid())
	m.blocks[b.Cid()] = b
}

func (m *impl) AddBlockPresence(p pair.CidChannelPair, t pb.Message_BlockPresenceType) {
	m.addBlockPresences(p.Cid, t, p.Channel)
}

func (m *impl) addBlockPresences(c cid.Cid, t pb.Message_BlockPresenceType,
	chns ...channel.Channel) {
	if _, ok := m.blocks[c]; ok {
		return
	}
	infos, ok := m.blockPresences[c]
	if !ok {
		infos = make(blockPresenceInfos)
		m.blockPresences[c] = infos
	}
	if _, ok := infos[t]; !ok {
		infos[t] = make(channelMap)
	}
	for _, chn := range chns {
		infos[t][chn] = struct{}{}
	}
}

func (m *impl) AddHave(p pair.CidChannelPair) {
	m.AddBlockPresence(p, pb.Message_Have)
}

func (m *impl) AddDontHave(p pair.CidChannelPair) {
	m.AddBlockPresence(p, pb.Message_DontHave)
}

func (m *impl) Size() int {
	size := 0
	for _, block := range m.blocks {
		size += len(block.RawData())
	}
	size += m.BlockPresencesSize()
	for _, e := range m.wantlist {
		size += e.Size()
	}

	return size
}

func (m *impl) BlockPresencesSize() int {
	size := 0
	for c, infos := range m.blockPresences {
		for t, chnmap := range infos {
			size += (&pb.Message_BlockPresence{
				Cid:      pb.Cid{Cid: c},
				Type:     t,
				Channels: chnmap.toChannelStrings(),
			}).Size()
		}
	}
	return size
}

// FromNet generates a new BitswapMessage from incoming data on an io.Reader.
func FromNet(r io.Reader) (BitSwapMessage, error) {
	reader := msgio.NewVarintReaderSize(r, network.MessageSizeMax)
	return FromMsgReader(reader)
}

// FromPBReader generates a new Bitswap message from a gogo-protobuf reader
func FromMsgReader(r msgio.Reader) (BitSwapMessage, error) {
	msg, err := r.ReadMsg()
	if err != nil {
		return nil, err
	}

	var pb pb.Message
	err = pb.Unmarshal(msg)
	r.ReleaseMsg(msg)
	if err != nil {
		return nil, err
	}

	return newMessageFromProto(pb)
}

func (m *impl) ToProtoV0() *pb.Message {
	pbm := new(pb.Message)
	pbm.Wantlist.Entries = make([]pb.Message_Wantlist_Entry, 0, len(m.wantlist))
	for _, e := range m.wantlist {
		pbm.Wantlist.Entries = append(pbm.Wantlist.Entries, e.ToPB())
	}
	pbm.Wantlist.Full = m.full

	blocks := m.Blocks()
	pbm.Blocks = make([][]byte, 0, len(blocks))
	for _, b := range blocks {
		pbm.Blocks = append(pbm.Blocks, b.RawData())
	}
	return pbm
}

func (m *impl) ToProtoV1() *pb.Message {
	pbm := new(pb.Message)
	pbm.Wantlist.Entries = make([]pb.Message_Wantlist_Entry, 0, len(m.wantlist))
	for _, e := range m.wantlist {
		pbm.Wantlist.Entries = append(pbm.Wantlist.Entries, e.ToPB())
	}
	pbm.Wantlist.Full = m.full

	blocks := m.Blocks()
	pbm.Payload = make([]pb.Message_Block, 0, len(blocks))
	for _, b := range blocks {
		pbm.Payload = append(pbm.Payload, pb.Message_Block{
			Data:   b.RawData(),
			Prefix: b.Cid().Prefix().Bytes(),
		})
	}

	pbm.BlockPresences = make([]pb.Message_BlockPresence, 0, len(m.blockPresences))
	for c, infos := range m.blockPresences {
		for t, chnmap := range infos {
			pbm.BlockPresences = append(pbm.BlockPresences, pb.Message_BlockPresence{
				Cid:      pb.Cid{Cid: c},
				Type:     t,
				Channels: chnmap.toChannelStrings(),
			})
		}
	}

	pbm.PendingBytes = m.PendingBytes()

	return pbm
}

func (m *impl) ToNetV0(w io.Writer) error {
	return write(w, m.ToProtoV0())
}

func (m *impl) ToNetV1(w io.Writer) error {
	return write(w, m.ToProtoV1())
}

func write(w io.Writer, m *pb.Message) error {
	size := m.Size()

	buf := pool.Get(size + binary.MaxVarintLen64)
	defer pool.Put(buf)

	n := binary.PutUvarint(buf, uint64(size))

	written, err := m.MarshalTo(buf[n:])
	if err != nil {
		return err
	}
	n += written

	_, err = w.Write(buf[:n])
	return err
}

func (m *impl) Loggable() map[string]interface{} {
	blocks := make([]string, 0, len(m.blocks))
	for _, v := range m.blocks {
		blocks = append(blocks, v.Cid().String())
	}
	return map[string]interface{}{
		"blocks": blocks,
		"wants":  m.Wantlist(),
	}
}
