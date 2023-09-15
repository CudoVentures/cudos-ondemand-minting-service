package account

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/simapp/params"
	encodingconfig "github.com/CudoVentures/cudos-ondemand-minting-service/internal/encoding_config"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestShouldQueryInfo(t *testing.T) {
	encodingConfig := encodingconfig.MakeEncodingConfig()
	accInfoClient := NewAccountInfoClient(nil, &encodingConfig)
	accInfoClient.authClient = &mockAuthInfoClient{}
	accInfo, err := accInfoClient.QueryInfo(context.Background(), "some address")
	require.NoError(t, err)
	require.Equal(t, model.AccountInfo{AccountNumber: 1, AccountSequence: 1}, accInfo)
}

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

func (maic *mockAuthInfoClient) AccountAddressByID(c1tx context.Context, in *auth.QueryAccountAddressByIDRequest, opts ...grpc.CallOption) (*auth.QueryAccountAddressByIDResponse, error) {
	return nil, nil
}

func (maic *mockAuthInfoClient) Account(ctx context.Context, in *auth.QueryAccountRequest, opts ...grpc.CallOption) (*auth.QueryAccountResponse, error) {
	if maic.accountQueryError != nil {
		return nil, maic.accountQueryError
	}

	acc, err := codectypes.NewAnyWithValue(auth.NewBaseAccount(types.AccAddress{}, &secp256k1.PubKey{}, 1, 1))
	if err != nil {
		return nil, err
	}

	return &auth.QueryAccountResponse{Account: acc}, nil
}

func (maic *mockAuthInfoClient) Params(ctx context.Context, in *auth.QueryParamsRequest, opts ...grpc.CallOption) (*auth.QueryParamsResponse, error) {
	return nil, nil
}

func (maic *mockAuthInfoClient) ModuleAccounts(ctx context.Context, in *auth.QueryModuleAccountsRequest, opts ...grpc.CallOption) (*auth.QueryModuleAccountsResponse, error) {
	return nil, nil
}

func (maic *mockAuthInfoClient) ModuleAccountByName(ctx context.Context, in *auth.QueryModuleAccountByNameRequest, opts ...grpc.CallOption) (*auth.QueryModuleAccountByNameResponse, error) {
	return nil, nil
}

func (maic *mockAuthInfoClient) Bech32Prefix(ctx context.Context, in *auth.Bech32PrefixRequest, opts ...grpc.CallOption) (*auth.Bech32PrefixResponse, error) {
	return nil, nil
}

func (maic *mockAuthInfoClient) AddressBytesToString(ctx context.Context, in *auth.AddressBytesToStringRequest, opts ...grpc.CallOption) (*auth.AddressBytesToStringResponse, error) {
	return nil, nil
}

func (maic *mockAuthInfoClient) AddressStringToBytes(ctx context.Context, in *auth.AddressStringToBytesRequest, opts ...grpc.CallOption) (*auth.AddressStringToBytesResponse, error) {
	return nil, nil
}

func (main *mockAuthInfoClient) AccountInfo(ctx context.Context, in *auth.QueryAccountInfoRequest, opts ...grpc.CallOption) (*auth.QueryAccountInfoResponse, error) {
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

func (mir *mockInterfaceRegistry) EnsureRegistered(iface interface{}) error {
	return nil
}

func (mir *mockInterfaceRegistry) Resolve(typeUrl string) (proto.Message, error) {
	return nil, nil
}

func (mir *mockInterfaceRegistry) UnpackAny(any *codectypes.Any, iface interface{}) error {
	return failedAccResponseUnpack
}

var failedAccRequest = errors.New("failed account request")
var failedAccResponseUnpack = errors.New("failed to unpack acc response")
