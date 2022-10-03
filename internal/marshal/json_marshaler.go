package marshal

import (
	"encoding/json"
)

func NewJsonMarshaler() *jsonMarshaler {
	return &jsonMarshaler{}
}

func (jm *jsonMarshaler) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (jm *jsonMarshaler) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

type jsonMarshaler struct {
}
