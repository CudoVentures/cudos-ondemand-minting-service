package state

import (
	"errors"
	"os"
	"testing"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
	"github.com/stretchr/testify/require"
)

func TestShouldFailToUpdateStateIfMarshalingFails(t *testing.T) {
	os.Remove(DefaultStateFilePath)
	fstate := NewFileState()
	fstate.marshaler = &failingMarshaler{}
	require.Equal(t, errors.New("failed to marshal"), fstate.UpdateState(model.State{}))
}

func TestShouldFailToUpdateStateIfStateFileDoesNotExists(t *testing.T) {
	os.Remove(DefaultStateFilePath)
	fstate := NewFileState()
	require.Error(t, fstate.UpdateState(model.State{}))
}

func TestShouldFailToGetStateIfUnmarshalingFails(t *testing.T) {
	fstate := NewFileState()
	fstate.CreateStateFileIfNotExists(0)
	fstate.marshaler = &failingMarshaler{}
	_, err := fstate.GetState()
	require.Equal(t, errors.New("failed to unmarshal"), err)
	os.Remove(DefaultStateFilePath)
}

func TestShouldFailToGetStateIfStateFileDoesNotExists(t *testing.T) {
	os.Remove(DefaultStateFilePath)
	fstate := NewFileState()
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
