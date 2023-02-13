package autorelay

import (
	"github.com/webtransport/libp2p-go/core/host"
)

type AutoRelayHost struct {
	host.Host
	ar *AutoRelay
}

func (h *AutoRelayHost) Close() error {
	_ = h.ar.Close()
	return h.Host.Close()
}

func NewAutoRelayHost(h host.Host, ar *AutoRelay) *AutoRelayHost {
	return &AutoRelayHost{Host: h, ar: ar}
}
