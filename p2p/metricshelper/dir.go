package metricshelper

import "github.com/webtransport/libp2p-go/core/network"

func GetDirection(dir network.Direction) string {
	switch dir {
	case network.DirOutbound:
		return "outbound"
	case network.DirInbound:
		return "inbound"
	default:
		return "unknown"
	}
}
