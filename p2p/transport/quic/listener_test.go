package libp2pquic

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	tpt "github.com/libp2p/go-libp2p/core/transport"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

func newTransport(t *testing.T, rcmgr network.ResourceManager) tpt.Transport {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	key, err := ic.UnmarshalRsaPrivateKey(x509.MarshalPKCS1PrivateKey(rsaKey))
	require.NoError(t, err)
	tr, err := NewTransport(key, nil, nil, rcmgr)
	require.NoError(t, err)
	return tr
}

func TestListenAddr(t *testing.T) {
	tr := newTransport(t, nil)
	defer tr.(io.Closer).Close()

	t.Run("for IPv4", func(t *testing.T) {
		localAddr := ma.StringCast("/ip4/127.0.0.1/udp/0/quic")
		ln, err := tr.Listen(localAddr)
		require.NoError(t, err)
		defer ln.Close()
		port := ln.Addr().(*net.UDPAddr).Port
		require.NotZero(t, port)
		var multiaddrsStrings []string
		for _, a := range ln.Multiaddrs() {
			multiaddrsStrings = append(multiaddrsStrings, a.String())
		}
		require.Contains(t, multiaddrsStrings, fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic", port))
		require.Contains(t, multiaddrsStrings, fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic-v1", port))
	})

	t.Run("for IPv6", func(t *testing.T) {
		localAddr := ma.StringCast("/ip6/::/udp/0/quic")
		ln, err := tr.Listen(localAddr)
		require.NoError(t, err)
		defer ln.Close()
		port := ln.Addr().(*net.UDPAddr).Port
		require.NotZero(t, port)
		var multiaddrsStrings []string
		for _, a := range ln.Multiaddrs() {
			multiaddrsStrings = append(multiaddrsStrings, a.String())
		}
		require.Contains(t, multiaddrsStrings, fmt.Sprintf("/ip6/::/udp/%d/quic", port))
		require.Contains(t, multiaddrsStrings, fmt.Sprintf("/ip6/::/udp/%d/quic-v1", port))
	})
}

func TestAccepting(t *testing.T) {
	tr := newTransport(t, nil)
	defer tr.(io.Closer).Close()
	ln, err := tr.Listen(ma.StringCast("/ip4/127.0.0.1/udp/0/quic"))
	require.NoError(t, err)
	done := make(chan struct{})
	go func() {
		ln.Accept()
		close(done)
	}()
	time.Sleep(100 * time.Millisecond)
	select {
	case <-done:
		t.Fatal("Accept didn't block")
	default:
	}
	require.NoError(t, ln.Close())
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Accept didn't return after the listener was closed")
	}
}

func TestAcceptAfterClose(t *testing.T) {
	tr := newTransport(t, nil)
	defer tr.(io.Closer).Close()
	ln, err := tr.Listen(ma.StringCast("/ip4/127.0.0.1/udp/0/quic"))
	require.NoError(t, err)
	require.NoError(t, ln.Close())
	_, err = ln.Accept()
	require.Error(t, err)
}
