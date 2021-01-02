package ws

import (
	"encoding/json"

	"github.com/google/uuid"
)

type DrinkRequest struct {
	DrinkerId   uuid.UUID
	DrinkerName string
	Request     []byte
}

func (dr DrinkRequest) ToBytes() ([]byte, error) {
	return json.Marshal(dr)
}
