package quicreuse

import (
	"net"
	"time"

	"github.com/lucas-clemente/quic-go"
)

var quicConfig = &quic.Config{
	MaxIncomingStreams:         256,
	MaxIncomingUniStreams:      -1,             // disable unidirectional streams
	MaxStreamReceiveWindow:     10 * (1 << 20), // 10 MB
	MaxConnectionReceiveWindow: 15 * (1 << 20), // 15 MB
	RequireAddressValidation: func(net.Addr) bool {
		// TODO(#1535): require source address validation when under load
		return false
	},
	KeepAlivePeriod: 15 * time.Second,
	Versions:        []quic.VersionNumber{quic.VersionDraft29, quic.Version1},
}
