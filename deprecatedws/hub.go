package ws

import (
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Hub struct {
	lgr     *zap.Logger
	Kettles map[uuid.UUID]*Kettle
}

func NewHub(lgr *zap.Logger) *Hub {
	return &Hub{
		lgr:     lgr,
		Kettles: make(map[uuid.UUID]*Kettle),
	}
}

func (h *Hub) AddKettle(lgr *zap.Logger, name string) *Kettle {
	kettle := NewKettle(lgr, name)
	h.Kettles[kettle.Id] = kettle
	go kettle.Run()
	return kettle
}

func (h *Hub) GetKettle(kettleId uuid.UUID) (*Kettle, bool) {
	// pretty trivial for now, but prob need to lock around map read/write.
	value, ok := h.Kettles[kettleId]
	return value, ok
}
