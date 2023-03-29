package state

import (
	"os"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/marshal"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
)

func NewFileState() *fileState {
	return &fileState{
		filePath:  DefaultStateFilePath,
		marshaler: marshal.NewJsonMarshaler(),
	}
}

func (s *fileState) GetState() (model.State, error) {
	fileData, err := os.ReadFile(s.filePath)
	if err != nil {
		return model.State{}, err
	}

	state := model.State{}
	if err := s.marshaler.Unmarshal([]byte(fileData), &state); err != nil {
		return model.State{}, err
	}

	return state, nil
}

func (s *fileState) UpdateState(state model.State) error {
	return s.updateState(state, false)
}

func (s *fileState) CreateStateFileIfNotExists(height int64) {
	exists, _ := s.checkIfStateFileExists()
	if !exists {
		s.updateState(model.State{
			Height: height,
		}, true)
	}
}

func (s *fileState) checkIfStateFileExists() (bool, error) {
	_, err := os.Stat(s.filePath)
	if os.IsNotExist(err) {
		return false, err
	}

	return true, nil
}

func (s *fileState) updateState(state model.State, createFile bool) error {
	fileData, err := s.marshaler.Marshal(state)
	if err != nil {
		return err
	}

	if !createFile {
		if _, err := s.checkIfStateFileExists(); err != nil {
			return err
		}
	}

	if err := os.WriteFile(s.filePath, fileData, 0644); err != nil {
		return err
	}

	return nil
}

var DefaultStateFilePath = "state.json"

type fileState struct {
	filePath  string
	marshaler marshaler
}

type marshaler interface {
	Unmarshal(data []byte, v any) error
	Marshal(v any) ([]byte, error)
}
