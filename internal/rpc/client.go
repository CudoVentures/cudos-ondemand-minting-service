package rpc

import (
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/client"
)

func (_ RPCConnector) MakeRPCClient(url string) (*rpchttp.HTTP, error) {
	return client.NewClientFromNode(url)
}

type RPCConnector struct {
}
