package rpc

import (
	"github.com/cosmos/cosmos-sdk/client"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
)

func (_ RPCConnector) MakeRPCClient(url string) (*rpchttp.HTTP, error) {
	return client.NewClientFromNode(url)
}

type RPCConnector struct {
}
