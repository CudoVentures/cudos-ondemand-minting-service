package state

import (
	"errors"
	"testing"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
	"github.com/stretchr/testify/require"
)

func TestShouldFailToUpdateStateIfMarshalingFails(t *testing.T) {
	fstate := NewFileState("./testdata/state.json")
	fstate.marshaler = &failingMarshaler{}
	require.Equal(t, errors.New("failed to marshal"), fstate.UpdateState(model.State{}))
}

func TestShouldFailToUpdateStateIfInvalidPath(t *testing.T) {
	fstate := NewFileState("invalid path\\/")
	require.Error(t, fstate.UpdateState(model.State{}))
}

func TestShouldFailToGetStateIfUnmarshalingFails(t *testing.T) {
	fstate := NewFileState("./testdata/state.json")
	fstate.marshaler = &failingMarshaler{}
	_, err := fstate.GetState()
	require.Equal(t, errors.New("failed to unmarshal"), err)
}

func TestShouldFailToGetStateIfInvalidPath(t *testing.T) {
	fstate := NewFileState("invalid path\\/")
	_, err := fstate.GetState()
	require.Error(t, err)
}

func (fm *failingMarshaler) Marshal(v any) ([]byte, error) {
	return nil, errors.New("failed to marshal")
}

func (fm *failingMarshaler) Unmarshal(data []byte, v any) error {
	return errors.New("failed to unmarshal")
}

type failingMarshaler struct {
}
