go-bitswap
==================

DAOT Labs's fork of [ipfs/go-bitswap](https://github.com/ipfs/go-bitswap).

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-DAOT%20Labs-red.svg?style=flat-square)](http://github.com/daotl)
[![Coverage Status](https://codecov.io/gh/daotl/go-bitswap/branch/master/graph/badge.svg)](https://codecov.io/gh/daotl/go-bitswap/branch/master)
[![Build Status](https://circleci.com/gh/daotl/go-bitswap.svg?style=svg)](https://circleci.com/gh/daotl/go-bitswap)

> An implementation of the bitswap protocol in Go!

## Lead Maintainer

[Nex](https://github.com/NexZhu)

## Table of Contents

- [Background](#background)
- [Install](#install)
- [Usage](#usage)
- [Implementation](#implementation)
- [Contribute](#contribute)
- [License](#license)

## Background

Bitswap is the data trading module for ipfs. It manages requesting and sending
blocks to and from other peers in the network. Bitswap has two main jobs:
- to acquire blocks requested by the client from the network
- to judiciously send blocks in its possession to other peers who want them

Bitswap is a message based protocol, as opposed to request-response. All messages
contain wantlists or blocks.

A node sends a wantlist to tell peers which blocks it wants. When a node receives
a wantlist it should check which blocks it has from the wantlist, and consider
sending the matching blocks to the requestor.

When a node receives blocks that it asked for, the node should send out a
notification called a 'Cancel' to tell its peers that the node no longer
wants those blocks.

`go-bitswap` provides an implementation of the Bitswap protocol in go.

[Learn more about how Bitswap works](./docs/how-bitswap-works.md)

## Install

`go-bitswap` requires Go >= 1.11 and can be installed using Go modules

## Usage

### Initializing a Bitswap Exchange

```golang
import (
  "context"
  bitswap "github.com/daotl/go-bitswap"
  bsnet "github.com/ipfs/go-graphsync/network"
  blockstore "github.com/daotl/go-ipfs-blockstore"
  "github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p-core/host"
)

var ctx context.Context
var host host.Host
var router routing.ContentRouting
var bstore blockstore.Blockstore

network := bsnet.NewFromIPFSHost(host, router)
exchange := bitswap.New(ctx, network, bstore)
```

Parameter Notes:

1. `ctx` is just the parent context for all of Bitswap
2. `network` is a network abstraction provided to Bitswap on top of libp2p & content routing. 
3. `bstore` is an IPFS blockstore

### Get A Block Synchronously

```golang
var c cid.Cid
var ctx context.Context
var exchange bitswap.Bitswap

block, err := exchange.GetBlock(ctx, c)
```

Parameter Notes:

1. `ctx` is the context for this request, which can be cancelled to cancel the request
2. `c` is the content ID of the block you're requesting

### Get Several Blocks Asynchronously

```golang
var cids []cid.Cid
var ctx context.Context
var exchange bitswap.Bitswap

blockChannel, err := exchange.GetBlocks(ctx, cids)
```

Parameter Notes:

1. `ctx` is the context for this request, which can be cancelled to cancel the request
2. `cids` is a slice of content IDs for the blocks you're requesting

### Get Related Blocks Faster With Sessions

In IPFS, content blocks are often connected to each other through a MerkleDAG. If you know ahead of time that block requests are related, Bitswap can make several optimizations internally in how it requests those blocks in order to get them faster. Bitswap provides a mechanism called a Bitswap Session to manage a series of block requests as part of a single higher level operation. You should initialize a Bitswap Session any time you intend to make a series of block requests that are related -- and whose responses are likely to come from the same peers.

```golang
var ctx context.Context
var cids []cids.cid
var exchange bitswap.Bitswap

session := exchange.NewSession(ctx)
blocksChannel, err := session.GetBlocks(ctx, cids)
// later
var relatedCids []cids.cid
relatedBlocksChannel, err := session.GetBlocks(ctx, relatedCids)
```

Note that `NewSession` returns an interface with `GetBlock` and `GetBlocks` methods that have the same signature as the overall Bitswap exchange.

### Tell bitswap a new block was added to the local datastore

```golang
var blk blocks.Block
var exchange bitswap.Bitswap

err := exchange.HasBlock(blk)
```

### Access Control

This fork additionally supports access control feature.

`Channel` from `github.com/daotl/go-ipfs-exchange-ipfs` represents an exchange
channel within which blocks can be exchanged possibly with access control.

In additional to the original APIs in the above examples which target the default
public channel, this fork added a new set of APIs which target a specific exchange channel:
- `GetBlockFromChannel`
- `WantlistForPeerAndChannel`
- `GetBlocksFromChannel`
- `HasBlockInChannel`
- `GetWantlistForChannel`
- `GetWantBlocksForChannel`
- `GetWantHavesForChannel`

Bitswap can be configured with `WithACFilter` to use an `ACFilter`:

```type ACFilter func(pid peer.ID, channel exchange.Channel, id cid.Cid) (bool, error)```

Then Bitswap will call configured `ACFilter` at the following lifecycle points: 
`afterIssueRequest`, `beforeSendWant`, `afterReceivedWant`, `beforeSendResponse`, `afterReceivedBlock` 
to determine if the requesting peer has the permission to access the specified
block in the specified exchange channel.

## Contribute

PRs are welcome!

Small note: If editing the Readme, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

MIT

Copyright for portions of this fork are held by Juan Batiz-Benet as part of the original
[go-bitswap](https://github.com/ipfs/go-bitswap) project. All other copyright for this
fork are held by DAOT Labs. All rights reserved.
