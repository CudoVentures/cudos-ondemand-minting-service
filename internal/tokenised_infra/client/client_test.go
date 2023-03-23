package client

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/marshal"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestShouldFailGetNFTDataWithInvalidUrl(t *testing.T) {
	client := NewTokenisedInfraClient(badUrl, marshal.NewJsonMarshaler())
	_, err := client.GetNFTData(context.Background(), config.Config{}, "testuid", "address", sdk.NewCoin("acudos", sdk.NewIntFromUint64(300)))
	require.Error(t, err)
}

func TestShouldFailGetNFTDataWithNotRunningService(t *testing.T) {
	client := NewTokenisedInfraClient(localServiceUrl, marshal.NewJsonMarshaler())
	_, err := client.GetNFTData(context.Background(), config.Config{}, "testuid", "address", sdk.NewCoin("acudos", sdk.NewIntFromUint64(300)))
	require.Error(t, err)
}

func TestShouldFailGetNFTDataIfInvalidJSONResponse(t *testing.T) {
	listener, err := net.Listen("tcp", ":1314")
	require.NoError(t, err)
	defer listener.Close()

	ws := newWebServer(listener)
	go ws.Start()
	defer ws.server.Shutdown(context.Background())

	client := NewTokenisedInfraClient(localServiceUrl, marshal.NewJsonMarshaler())
	_, err = client.GetNFTData(context.Background(), config.Config{}, "testuid", "address", sdk.NewCoin("acudos", sdk.NewIntFromUint64(300)))
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "invalid character"))
}

func TestGetNFTDataShouldReturnEmptyDataWithNoErrWhenNotUIDNotFound(t *testing.T) {
	listener, err := net.Listen("tcp", ":1314")
	require.NoError(t, err)
	defer listener.Close()

	ws := newWebServer(listener)
	go ws.Start()
	defer ws.server.Shutdown(context.Background())

	client := NewTokenisedInfraClient(localServiceUrl, marshal.NewJsonMarshaler())
	data, err := client.GetNFTData(context.Background(), config.Config{}, "notfounduid", "address", sdk.NewCoin("acudos", sdk.NewIntFromUint64(300)))
	require.NoError(t, err)
	require.Equal(t, model.NFTData{}, data)
}

func TestShouldFailParseBodyWithBodyError(t *testing.T) {
	client := NewTokenisedInfraClient(localServiceUrl, marshal.NewJsonMarshaler())
	readCloser := mockReadCloser{}
	readCloser.On("Read", mock.AnythingOfType("[]uint8")).Return(0, errors.New("error reading"))
	readCloser.On("Close").Return(errors.New("error closing"))
	_, err := client.parseBody(&http.Response{
		Body: &readCloser,
	})
	require.Error(t, err)
}

// func TestShouldFailMarkMintedNFTWithInvalidUrl(t *testing.T) {
// 	client := NewTokenisedInfraClient(badUrl, marshal.NewJsonMarshaler())
// 	require.Error(t, client.MarkMintedNFT(context.Background(), "txhash", "testuid"))
// }

// func TestShouldFailMarkMintedNFTWithNotRunningService(t *testing.T) {
// 	client := NewTokenisedInfraClient(localServiceUrl, marshal.NewJsonMarshaler())
// 	require.Error(t, client.MarkMintedNFT(context.Background(), "txhash", "testuid"))
// }

// func TestShouldFailMarkMintedNFTIfFailsToMarshal(t *testing.T) {
// 	client := NewTokenisedInfraClient(localServiceUrl, &mockMarshaler{})
// 	require.Equal(t, failedMarshal, client.MarkMintedNFT(context.Background(), "txhash", "testuid", "address", sdk.NewCoin("acudos", sdk.NewIntFromUint64(300))))
// }

// func (mm *mockMarshaler) Marshal(v any) ([]byte, error) {
// 	return nil, failedMarshal
// }

// var failedMarshal = errors.New("failed to marshal")

// func (mm *mockMarshaler) Unmarshal(data []byte, v any) error {
// 	return nil
// }

// type mockMarshaler struct {
// }

func TestShouldParseBody(t *testing.T) {
	client := NewTokenisedInfraClient(localServiceUrl, marshal.NewJsonMarshaler())
	bz, err := client.marshaler.Marshal(&model.NFTData{Id: "1"})
	require.NoError(t, err)

	nftData, err := client.parseBody(&http.Response{
		Body: io.NopCloser(strings.NewReader(string(bz))),
	})
	require.NoError(t, err)
	require.EqualValues(t, model.NFTData{Id: "1", Price: sdk.ZeroInt()}, nftData)
}

type webServer struct {
	server   http.Server
	listener net.Listener
}

func newWebServer(listener net.Listener) *webServer {
	return &webServer{
		server:   http.Server{},
		listener: listener,
	}
}

func (ws *webServer) Start() {
	ws.server.Handler = ws
	ws.server.Serve(ws.listener)
}

func (ws *webServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, "testuid") {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

type mockReadCloser struct {
	mock.Mock
}

func (mrc *mockReadCloser) Read(p []byte) (n int, err error) {
	args := mrc.Called(p)
	return args.Int(0), args.Error(1)
}

func (mrc *mockReadCloser) Close() error {
	args := mrc.Called()
	return args.Error(0)
}

const (
	badUrl          = ":badurl"
	localServiceUrl = "http://127.0.0.1:1314"
)
