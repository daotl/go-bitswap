package message

import (
	"github.com/daotl/go-bitswap/wantlist"
	exchange "github.com/daotl/go-ipfs-exchange-interface"
	blocks "github.com/ipfs/go-block-format"
)

type MsgBlock interface {
	blocks.Block
	GetChannel() exchange.Channel
	GetKey() wantlist.WantKey
	GetBlock() blocks.Block
}

type msgBlock struct {
	blocks.Block
	ch exchange.Channel
}

func NewMsgBlock(block blocks.Block, ch exchange.Channel) MsgBlock {
	return &msgBlock{Block: block, ch: ch}
}

func (m *msgBlock) GetChannel() exchange.Channel {
	return m.ch
}

func (m *msgBlock) GetKey() wantlist.WantKey {
	return wantlist.NewWantKey(m.Cid(), m.ch)
}

func (m *msgBlock) GetBlock() blocks.Block {
	return m.Block
}

func BlocksToMsgBlocks(blks []blocks.Block, ch exchange.Channel) []MsgBlock {
	msgBlks := make([]MsgBlock, 0, len(blks))
	for _, blk := range blks {
		msgBlks = append(msgBlks, NewMsgBlock(blk, ch))
	}
	return msgBlks
}

func MsgBlocksToBlocks(msgBlks []MsgBlock) []blocks.Block {
	blks := make([]blocks.Block, 0, len(msgBlks))
	for _, blk := range msgBlks {
		blks = append(blks, blk)
	}
	return blks
}
