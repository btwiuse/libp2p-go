// Package core provides convenient access to foundational, central go-libp2p primitives via type aliases.
package core

import (
	"github.com/webtransport/libp2p-go/core/host"
	"github.com/webtransport/libp2p-go/core/network"
	"github.com/webtransport/libp2p-go/core/peer"
	"github.com/webtransport/libp2p-go/core/protocol"

	"github.com/multiformats/go-multiaddr"
)

// Multiaddr aliases the Multiaddr type from github.com/multiformats/go-multiaddr.
//
// Refer to the docs on that type for more info.
type Multiaddr = multiaddr.Multiaddr

// PeerID aliases peer.ID.
//
// Refer to the docs on that type for more info.
type PeerID = peer.ID

// ProtocolID aliases protocol.ID.
//
// Refer to the docs on that type for more info.
type ProtocolID = protocol.ID

// PeerAddrInfo aliases peer.AddrInfo.
//
// Refer to the docs on that type for more info.
type PeerAddrInfo = peer.AddrInfo

// Host aliases host.Host.
//
// Refer to the docs on that type for more info.
type Host = host.Host

// Network aliases network.Network.
//
// Refer to the docs on that type for more info.
type Network = network.Network

// Conn aliases network.Conn.
//
// Refer to the docs on that type for more info.
type Conn = network.Conn

// Stream aliases network.Stream.
//
// Refer to the docs on that type for more info.
type Stream = network.Stream
