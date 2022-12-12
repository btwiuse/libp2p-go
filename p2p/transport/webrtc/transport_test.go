package libp2pwebrtc

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	tpt "github.com/libp2p/go-libp2p/core/transport"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func getTransport(t *testing.T) (tpt.Transport, peer.ID) {
	privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	require.NoError(t, err)
	rcmgr := &network.NullResourceManager{}
	transport, err := New(privKey, rcmgr)
	require.NoError(t, err)
	peerID, err := peer.IDFromPrivateKey(privKey)
	require.NoError(t, err)
	return transport, peerID
}

var (
	listenerIp net.IP
	dialerIp   net.IP
)

func TestMain(m *testing.M) {
	listenerIp, dialerIp = getListenerAndDialerIP()
	os.Exit(m.Run())
}

func TestTransportWebRTC_CanDial(t *testing.T) {
	tr, _ := getTransport(t)
	invalid := []string{
		"/ip4/1.2.3.4/udp/1234/webrtc",
		"/dns/test.test/udp/1234/webrtc",
		"/dns/test.test/udp/1234/webrtc/certhash/uEiAsGPzpiPGQzSlVHRXrUCT5EkTV7YFrV4VZ3hpEKTd_zg",
	}

	valid := []string{
		"/ip4/1.2.3.4/udp/1234/webrtc/certhash/uEiAsGPzpiPGQzSlVHRXrUCT5EkTV7YFrV4VZ3hpEKTd_zg",
		"/ip6/0:0:0:0:0:0:0:1/udp/1234/webrtc/certhash/uEiAsGPzpiPGQzSlVHRXrUCT5EkTV7YFrV4VZ3hpEKTd_zg",
		"/ip6/::1/udp/1234/webrtc/certhash/uEiAsGPzpiPGQzSlVHRXrUCT5EkTV7YFrV4VZ3hpEKTd_zg",
	}

	for _, addr := range invalid {
		ma, err := multiaddr.NewMultiaddr(addr)
		require.NoError(t, err)
		require.Equal(t, false, tr.CanDial(ma))
	}

	for _, addr := range valid {
		ma, err := multiaddr.NewMultiaddr(addr)
		require.NoError(t, err)
		require.Equal(t, true, tr.CanDial(ma), addr)
	}
}

func TestTransportWebRTC_ListenFailsOnNonWebRTCMultiaddr(t *testing.T) {
	tr, _ := getTransport(t)
	testAddrs := []string{
		"/ip4/0.0.0.0/udp/0",
		"/ip4/0.0.0.0/tcp/0/wss",
	}
	for _, addr := range testAddrs {
		listenMultiaddr, err := multiaddr.NewMultiaddr(addr)
		require.NoError(t, err)
		listener, err := tr.Listen(listenMultiaddr)
		require.Error(t, err)
		require.Nil(t, listener)
	}
}

func TestTransportWebRTC_DialFailsOnUnsupportedHashFunction(t *testing.T) {
	tr, _ := getTransport(t)
	hash := sha3.New512()
	certhash := func() string {
		_, err := hash.Write([]byte("test-data"))
		require.NoError(t, err)
		mh, err := multihash.Encode(hash.Sum([]byte{}), multihash.SHA3_512)
		require.NoError(t, err)
		certhash, err := multibase.Encode(multibase.Base58BTC, mh)
		require.NoError(t, err)
		return certhash
	}()
	testaddr, err := multiaddr.NewMultiaddr("/ip4/1.2.3.4/udp/1234/webrtc/certhash/" + certhash)
	require.NoError(t, err)
	_, err = tr.Dial(context.Background(), testaddr, "")
	require.ErrorContains(t, err, "unsupported hash function")
}

func TestTransportWebRTC_CanListenSingle(t *testing.T) {
	tr, listeningPeer := getTransport(t)
	tr1, connectingPeer := getTransport(t)
	listenMultiaddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/0/webrtc", listenerIp))
	require.NoError(t, err)
	listener, err := tr.Listen(listenMultiaddr)
	require.NoError(t, err)

	go func() {
		_, err := tr1.Dial(context.Background(), listener.Multiaddr(), listeningPeer)
		require.NoError(t, err)
	}()

	conn, err := listener.Accept()
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, connectingPeer, conn.RemotePeer())
}

func TestTransportWebRTC_CanListenMultiple(t *testing.T) {
	tr, listeningPeer := getTransport(t)
	listenMultiaddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/0/webrtc", listenerIp))
	require.NoError(t, err)
	listener, err := tr.Listen(listenMultiaddr)
	require.NoError(t, err)
	count := 5

	for i := 0; i < count; i++ {
		go func() {
			ctr, _ := getTransport(t)
			conn, err := ctr.Dial(context.Background(), listener.Multiaddr(), listeningPeer)
			require.NoError(t, err)
			require.Equal(t, conn.RemotePeer(), listeningPeer)
		}()
	}

	for i := 0; i < count; i++ {
		_, err := listener.Accept()
		require.NoError(t, err)
		t.Logf("listener accepted connection: %d", i)
	}
}

func TestTransportWebRTC_CanCreateSuccessiveConnections(t *testing.T) {
	tr, listeningPeer := getTransport(t)
	listenMultiaddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/0/webrtc", listenerIp))
	require.NoError(t, err)
	listener, err := tr.Listen(listenMultiaddr)
	require.NoError(t, err)
	count := 5

	go func() {
		for i := 0; i < count; i++ {
			ctr, _ := getTransport(t)
			conn, err := ctr.Dial(context.Background(), listener.Multiaddr(), listeningPeer)
			require.NoError(t, err)
			require.Equal(t, conn.RemotePeer(), listeningPeer)
			conn.Close()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	for i := 0; i < count; i++ {
		_, err := listener.Accept()
		require.NoError(t, err)
	}
}

func TestTransportWebRTC_ListenerCanCreateStreams(t *testing.T) {
	tr, listeningPeer := getTransport(t)
	tr1, connectingPeer := getTransport(t)
	listenMultiaddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/0/webrtc", listenerIp))
	require.NoError(t, err)
	listener, err := tr.Listen(listenMultiaddr)
	require.NoError(t, err)

	streamChan := make(chan network.MuxedStream)
	go func() {
		conn, err := tr1.Dial(context.Background(), listener.Multiaddr(), listeningPeer)
		require.NoError(t, err)
		t.Logf("connection opened by dialer")
		stream, err := conn.AcceptStream()
		require.NoError(t, err)
		t.Logf("dialer accepted stream")
		streamChan <- stream
	}()

	conn, err := listener.Accept()
	require.NoError(t, err)
	t.Logf("listener accepted connection")
	require.Equal(t, connectingPeer, conn.RemotePeer())

	stream, err := conn.OpenStream(context.Background())
	require.NoError(t, err)
	t.Logf("listener opened stream")
	_, err = stream.Write([]byte("test"))
	require.NoError(t, err)

	var str network.MuxedStream
	select {
	case str = <-streamChan:
	case <-time.After(3 * time.Second):
		t.Fatal("stream opening timed out")
	}
	buf := make([]byte, 100)
	stream.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := str.Read(buf)
	require.NoError(t, err)
	require.Equal(t, "test", string(buf[:n]))

}

func TestTransportWebRTC_DialerCanCreateStreams(t *testing.T) {
	tr, listeningPeer := getTransport(t)
	listenMultiaddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/0/webrtc", listenerIp))
	require.NoError(t, err)
	listener, err := tr.Listen(listenMultiaddr)
	require.NoError(t, err)

	tr1, connectingPeer := getTransport(t)
	done := make(chan struct{})

	go func() {
		lconn, err := listener.Accept()
		require.NoError(t, err)
		require.Equal(t, connectingPeer, lconn.RemotePeer())

		stream, err := lconn.AcceptStream()
		require.NoError(t, err)
		buf := make([]byte, 100)
		n, err := stream.Read(buf)
		require.NoError(t, err)
		require.Equal(t, "test", string(buf[:n]))

		done <- struct{}{}
	}()

	go func() {
		conn, err := tr1.Dial(context.Background(), listener.Multiaddr(), listeningPeer)
		require.NoError(t, err)
		t.Logf("dialer opened connection")
		stream, err := conn.OpenStream(context.Background())
		require.NoError(t, err)
		t.Logf("dialer opened stream")
		_, err = stream.Write([]byte("test"))
		require.NoError(t, err)

	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out")
	}

}

func TestTransportWebRTC_StreamSetReadDeadline(t *testing.T) {
	tr, listeningPeer := getTransport(t)
	listenMultiaddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/0/webrtc", listenerIp))
	require.NoError(t, err)
	listener, err := tr.Listen(listenMultiaddr)
	require.NoError(t, err)

	tr1, connectingPeer := getTransport(t)

	go func() {
		lconn, err := listener.Accept()
		require.NoError(t, err)
		require.Equal(t, connectingPeer, lconn.RemotePeer())
		_, err = lconn.AcceptStream()
		require.NoError(t, err)
	}()

	conn, err := tr1.Dial(context.Background(), listener.Multiaddr(), listeningPeer)
	require.NoError(t, err)
	stream, err := conn.OpenStream(context.Background())
	require.NoError(t, err)

	// deadline set to the past
	timer := time.AfterFunc(150*time.Millisecond, func() {
		stream.SetReadDeadline(time.Now().Add(-200 * time.Millisecond))
	})
	defer timer.Stop()
	_, err = stream.Read([]byte{0, 0})
	require.ErrorIs(t, err, os.ErrDeadlineExceeded)

	// future deadline exceeded
	stream.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, err = stream.Read([]byte{0, 0})
	require.ErrorIs(t, err, os.ErrDeadlineExceeded)

}

func TestTransportWebRTC_StreamSetWriteDeadline(t *testing.T) {
	tr, listeningPeer := getTransport(t)
	listenMultiaddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/0/webrtc", listenerIp))
	require.NoError(t, err)
	listener, err := tr.Listen(listenMultiaddr)
	require.NoError(t, err)

	tr1, connectingPeer := getTransport(t)

	go func() {
		lconn, err := listener.Accept()
		require.NoError(t, err)
		require.Equal(t, connectingPeer, lconn.RemotePeer())
		_, err = lconn.AcceptStream()
		require.NoError(t, err)
	}()

	conn, err := tr1.Dial(context.Background(), listener.Multiaddr(), listeningPeer)
	require.NoError(t, err)
	stream, err := conn.OpenStream(context.Background())
	require.NoError(t, err)

	stream.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))
	largeBuffer := make([]byte, 2*1024*1024)
	_, err = stream.Write(largeBuffer)
	require.ErrorIs(t, err, os.ErrDeadlineExceeded)
}

func TestTransportWebRTC_StreamWriteBufferContention(t *testing.T) {
	tr, listeningPeer := getTransport(t)
	listenMultiaddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/0/webrtc", listenerIp))
	require.NoError(t, err)
	listener, err := tr.Listen(listenMultiaddr)
	require.NoError(t, err)

	tr1, connectingPeer := getTransport(t)

	for i := 0; i < 2; i++ {
		go func() {
			lconn, err := listener.Accept()
			require.NoError(t, err)
			require.Equal(t, connectingPeer, lconn.RemotePeer())
			_, err = lconn.AcceptStream()
			require.NoError(t, err)
		}()

	}

	conn, err := tr1.Dial(context.Background(), listener.Multiaddr(), listeningPeer)
	require.NoError(t, err)

	errC := make(chan error)
	// writers
	for i := 0; i < 2; i++ {
		go func() {
			stream, err := conn.OpenStream(context.Background())
			require.NoError(t, err)

			stream.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))
			largeBuffer := make([]byte, 2*1024*1024)
			_, err = stream.Write(largeBuffer)
			errC <- err
		}()
	}

	require.ErrorIs(t, <-errC, os.ErrDeadlineExceeded)
	require.ErrorIs(t, <-errC, os.ErrDeadlineExceeded)

}

func TestTransportWebRTC_ReadPartialMessage(t *testing.T) {
	tr, listeningPeer := getTransport(t)
	listenMultiaddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/0/webrtc", listenerIp))
	require.NoError(t, err)
	listener, err := tr.Listen(listenMultiaddr)
	require.NoError(t, err)

	tr1, connectingPeer := getTransport(t)

	go func() {
		lconn, err := listener.Accept()
		require.NoError(t, err)
		require.Equal(t, connectingPeer, lconn.RemotePeer())
		stream, err := lconn.AcceptStream()
		require.NoError(t, err)
		_, err = stream.Write(make([]byte, 2*1024*1024))
		require.NoError(t, err)
	}()

	conn, err := tr1.Dial(context.Background(), listener.Multiaddr(), listeningPeer)
	require.NoError(t, err)
	stream, err := conn.OpenStream(context.Background())
	require.NoError(t, err)

	buf := make([]byte, 10)
	stream.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err := stream.Read(buf)
	require.NoError(t, err)
	require.Equal(t, n, 10)

}

func TestTransportWebRTC_ReadZeroBytes(t *testing.T) {
	tr, listeningPeer := getTransport(t)
	listenMultiaddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/0/webrtc", listenerIp))
	require.NoError(t, err)
	listener, err := tr.Listen(listenMultiaddr)
	require.NoError(t, err)

	tr1, connectingPeer := getTransport(t)

	go func() {
		lconn, err := listener.Accept()
		require.NoError(t, err)
		require.Equal(t, connectingPeer, lconn.RemotePeer())
		stream, err := lconn.AcceptStream()
		require.NoError(t, err)
		_, err = stream.Write(make([]byte, 2*1024*1024))
		require.NoError(t, err)
	}()

	conn, err := tr1.Dial(context.Background(), listener.Multiaddr(), listeningPeer)
	require.NoError(t, err)
	stream, err := conn.OpenStream(context.Background())
	require.NoError(t, err)

	stream.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err := stream.Read([]byte{})
	require.NoError(t, err)
	require.Equal(t, n, 0)
}

func TestTransportWebRTC_StreamCanCloseWhenReadActive(t *testing.T) {
	tr, listeningPeer := getTransport(t)
	listenMultiaddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/0/webrtc", listenerIp))
	require.NoError(t, err)
	listener, err := tr.Listen(listenMultiaddr)
	require.NoError(t, err)

	tr1, connectingPeer := getTransport(t)
	done := make(chan struct{})

	go func() {
		lconn, err := listener.Accept()
		require.NoError(t, err)
		t.Logf("listener accepted connection")
		require.Equal(t, connectingPeer, lconn.RemotePeer())
		done <- struct{}{}
	}()

	conn, err := tr1.Dial(context.Background(), listener.Multiaddr(), listeningPeer)
	require.NoError(t, err)
	t.Logf("dialer opened connection")
	stream, err := conn.OpenStream(context.Background())
	require.NoError(t, err)

	time.AfterFunc(100*time.Millisecond, func() {
		err = stream.Close()
		require.NoError(t, err)
	})

	_, readerr := stream.Read(make([]byte, 19))
	require.ErrorIs(t, readerr, io.EOF)

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out")
	}
}

func TestTransportWebRTC_ReceiveFlagsAfterReadClosed(t *testing.T) {
	tr, listeningPeer := getTransport(t)
	listenMultiaddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/0/webrtc", listenerIp))
	require.NoError(t, err)
	listener, err := tr.Listen(listenMultiaddr)
	require.NoError(t, err)

	tr1, connectingPeer := getTransport(t)
	done := make(chan struct{})

	go func() {
		lconn, err := listener.Accept()
		require.NoError(t, err)
		t.Logf("listener accepted connection")
		require.Equal(t, connectingPeer, lconn.RemotePeer())
		stream, err := lconn.AcceptStream()
		require.NoError(t, err)
		n, err := stream.Read(make([]byte, 10))
		require.NoError(t, err)
		require.Equal(t, 10, n)
		// stop reader
		err = stream.Reset()
		require.NoError(t, err)
		done <- struct{}{}
	}()

	conn, err := tr1.Dial(context.Background(), listener.Multiaddr(), listeningPeer)
	require.NoError(t, err)
	t.Logf("dialer opened connection")
	stream, err := conn.OpenStream(context.Background())
	require.NoError(t, err)

	err = stream.CloseRead()
	require.NoError(t, err)
	_, err = stream.Read([]byte{0})
	require.ErrorIs(t, err, io.EOF)
	_, err = stream.Write(make([]byte, 10))
	require.NoError(t, err)
	<-done
	_, err = stream.Write(make([]byte, 2*1024*1024))
	require.ErrorContains(t, err, "closed")
}

func TestTransportWebRTC_PeerConnectionDTLSFailed(t *testing.T) {
	// test multihash
	encoded, err := hex.DecodeString("ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad")
	require.NoError(t, err)

	testMultihash := &multihash.DecodedMultihash{
		Code:   multihash.SHA2_256,
		Name:   multihash.Codes[multihash.SHA2_256],
		Digest: encoded,
		Length: len(encoded),
	}

	tr, listeningPeer := getTransport(t)
	listenMultiaddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/0/webrtc", listenerIp))
	require.NoError(t, err)
	listener, err := tr.Listen(listenMultiaddr)
	require.NoError(t, err)

	tr1, _ := getTransport(t)

	go func() {
		listener.Accept()
	}()

	badMultiaddr, _ := multiaddr.SplitFunc(listener.Multiaddr(), func(component multiaddr.Component) bool {
		return component.Protocol().Code == multiaddr.P_CERTHASH
	})

	encodedCerthash, err := multihash.Encode(testMultihash.Digest, testMultihash.Code)
	require.NoError(t, err)
	badEncodedCerthash, err := multibase.Encode(multibase.Base58BTC, encodedCerthash)
	require.NoError(t, err)
	badCerthash, err := multiaddr.NewMultiaddr(fmt.Sprintf("/certhash/%s", badEncodedCerthash))
	require.NoError(t, err)
	badMultiaddr = badMultiaddr.Encapsulate(badCerthash)

	conn, err := tr1.Dial(context.Background(), badMultiaddr, listeningPeer)
	require.Nil(t, conn)
	require.Error(t, err)

	require.ErrorContains(t, err, "failed")

}