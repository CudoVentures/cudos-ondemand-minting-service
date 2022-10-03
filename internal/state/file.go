package state

import (
	"io/ioutil"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/marshal"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
)

func NewFileState(filePath string) *fileState {
	return &fileState{
		filePath:  filePath,
		marshaler: marshal.NewJsonMarshaler(),
	}
}

func (s fileState) GetState() (model.State, error) {
	fileData, err := ioutil.ReadFile(s.filePath)
	if err != nil {
		return model.State{}, err
	}

	state := model.State{}
	if err := s.marshaler.Unmarshal([]byte(fileData), &state); err != nil {
		return model.State{}, err
	}

	return state, nil
}

func (s fileState) UpdateState(state model.State) error {
	fileData, err := s.marshaler.Marshal(state)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(s.filePath, fileData, 0); err != nil {
		return err
	}

	return nil
}

type fileState struct {
	filePath  string
	marshaler marshaler
}

type marshaler interface {
	Unmarshal(data []byte, v any) error
	Marshal(v any) ([]byte, error)
}
