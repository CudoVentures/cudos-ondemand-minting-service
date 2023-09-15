package account

import (
	"context"

	"cosmossdk.io/simapp/params"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/grpc"
)

func NewAccountInfoClient(grpcConn *grpc.ClientConn, encodingConfig *params.EncodingConfig) *accountInfoClient {
	return &accountInfoClient{
		encodingConfig: encodingConfig,
		authClient:     auth.NewQueryClient(grpcConn),
	}
}

func (aic *accountInfoClient) QueryInfo(ctx context.Context, address string) (model.AccountInfo, error) {
	res, err := aic.authClient.Account(ctx, &auth.QueryAccountRequest{Address: address})
	if err != nil {
		return model.AccountInfo{}, err
	}

	var accountInfo types.AccountI
	if err := aic.encodingConfig.InterfaceRegistry.UnpackAny(res.Account, &accountInfo); err != nil {
		return model.AccountInfo{}, err
	}

	return model.AccountInfo{
		AccountNumber:   accountInfo.GetAccountNumber(),
		AccountSequence: accountInfo.GetSequence(),
	}, nil
}

type accountInfoClient struct {
	encodingConfig *params.EncodingConfig
	authClient     auth.QueryClient
}
