package account

import (
	"context"
	"errors"
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestShouldFailIfAuthClientAccRequestFails(t *testing.T) {
	accInfoClient := NewAccountInfoClient(nil, nil)
	accInfoClient.authClient = &mockAuthInfoClient{accountQueryError: failedAccRequest}
	_, err := accInfoClient.QueryInfo(context.Background(), "some address")
	require.Equal(t, failedAccRequest, err)
}

func TestShouldFailIfUnpackFails(t *testing.T) {
	accInfoClient := NewAccountInfoClient(nil, nil)
	accInfoClient.authClient = &mockAuthInfoClient{}
	encodingConfig := params.EncodingConfig{}
	encodingConfig.InterfaceRegistry = &mockInterfaceRegistry{}
	accInfoClient.encodingConfig = &encodingConfig
	_, err := accInfoClient.QueryInfo(context.Background(), "some address")
	require.Equal(t, failedAccResponseUnpack, err)
}

type mockAuthInfoClient struct {
	accountQueryError error
}

func (maic *mockAuthInfoClient) Accounts(ctx context.Context, in *auth.QueryAccountsRequest, opts ...grpc.CallOption) (*auth.QueryAccountsResponse, error) {
	return nil, nil
}

func (maic *mockAuthInfoClient) Account(ctx context.Context, in *auth.QueryAccountRequest, opts ...grpc.CallOption) (*auth.QueryAccountResponse, error) {
	if maic.accountQueryError != nil {
		return nil, maic.accountQueryError
	}
	return &auth.QueryAccountResponse{}, nil
}

func (maic *mockAuthInfoClient) Params(ctx context.Context, in *auth.QueryParamsRequest, opts ...grpc.CallOption) (*auth.QueryParamsResponse, error) {
	return nil, nil
}

type mockInterfaceRegistry struct {
}

func (mir *mockInterfaceRegistry) RegisterInterface(protoName string, iface interface{}, impls ...proto.Message) {
}

func (mir *mockInterfaceRegistry) RegisterImplementations(iface interface{}, impls ...proto.Message) {
}

func (mir *mockInterfaceRegistry) ListAllInterfaces() []string {
	return []string{}
}

func (mir *mockInterfaceRegistry) ListImplementations(ifaceTypeURL string) []string {
	return []string{}
}

func (mir *mockInterfaceRegistry) Resolve(typeUrl string) (proto.Message, error) {
	return nil, nil
}

func (mir *mockInterfaceRegistry) UnpackAny(any *codectypes.Any, iface interface{}) error {
	return failedAccResponseUnpack
}

var failedAccRequest = errors.New("failed account request")
var failedAccResponseUnpack = errors.New("failed to unpack acc response")
