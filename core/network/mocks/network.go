package mocknetwork

//go:generate sh -c "go run github.com/golang/mock/mockgen -package mocknetwork -destination mock_resource_manager.go github.com/webtransport/libp2p-go/core/network ResourceManager"
//go:generate sh -c "go run github.com/golang/mock/mockgen -package mocknetwork -destination mock_conn_management_scope.go github.com/webtransport/libp2p-go/core/network ConnManagementScope"
//go:generate sh -c "go run github.com/golang/mock/mockgen -package mocknetwork -destination mock_stream_management_scope.go github.com/webtransport/libp2p-go/core/network StreamManagementScope"
//go:generate sh -c "go run github.com/golang/mock/mockgen -package mocknetwork -destination mock_peer_scope.go github.com/webtransport/libp2p-go/core/network PeerScope"
//go:generate sh -c "go run github.com/golang/mock/mockgen -package mocknetwork -destination mock_protocol_scope.go github.com/webtransport/libp2p-go/core/network ProtocolScope"
