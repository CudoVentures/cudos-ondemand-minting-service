package state

import (
	"encoding/json"
	"io/ioutil"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
)

func NewFileState(filePath string) *fileState {
	return &fileState{
		filePath: filePath,
	}
}

func (s fileState) GetState() (model.State, error) {
	fileData, err := ioutil.ReadFile(s.filePath)
	if err != nil {
		return model.State{}, err
	}

	state := model.State{}
	if err := json.Unmarshal([]byte(fileData), &state); err != nil {
		return model.State{}, err
	}

	return state, nil
}

func (s fileState) UpdateState(state model.State) error {
	fileData, err := json.Marshal(state)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(s.filePath, fileData, 0); err != nil {
		return err
	}

	return nil
}

type fileState struct {
	filePath string
}
