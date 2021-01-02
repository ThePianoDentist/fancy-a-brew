package ws

import (
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Kettle struct {
	Id   uuid.UUID
	lgr  *zap.Logger
	name string

	// Registered drinkers.
	drinkers map[uuid.UUID]*Drinker

	// maybe a substructure related to current-round (can keep track of what made?)
	currentMaker          *Drinker
	currentRoundTimestamp time.Time

	// Inbound messages from the drinkers.
	drinkRequests chan DrinkRequest

	// TODO offers need to be "completed". or timed out. "completion" with a 15 min time-out seems sensible
	// or a 5-min "wheres my drink muthafucka?"
	drinkOffers chan uuid.UUID

	roundCompleted chan struct{}

	// Register requests from the drinkers.
	register chan *Drinker

	// Unregister requests from drinkers.
	unregister chan uuid.UUID
}

func NewKettle(lgr *zap.Logger, name string) *Kettle {
	return &Kettle{
		Id:   uuid.New(),
		lgr:  lgr,
		name: name,
		// do buffers matter. need to analyse concurrent behaviour of app
		drinkRequests:  make(chan DrinkRequest, 16),
		drinkOffers:    make(chan uuid.UUID),
		roundCompleted: make(chan struct{}),
		register:       make(chan *Drinker),
		unregister:     make(chan uuid.UUID),
		drinkers:       make(map[uuid.UUID]*Drinker),
	}
}

func (k *Kettle) Run() {
	for {
		select {
		case drinker := <-k.register:
			k.drinkers[drinker.Id] = drinker
		case drinkerId := <-k.unregister:
			if drinker, ok := k.drinkers[drinkerId]; ok {
				delete(k.drinkers, drinkerId)
				close(drinker.send)
			}
			// Important if nobody listening to kettle anymore. We should finish this kettle running goroutine.
			// Is there any state that needs recording for if re-open?
			if len(k.drinkers) == 0 {
				return
			}
		case drinkerId := <-k.drinkOffers:
			k.processDrinkOffer(drinkerId)
		case dr := <-k.drinkRequests:
			k.processDrinkRequest(dr)
		case <-k.roundCompleted:
			k.currentRoundTimestamp = time.Time{}
			k.currentMaker = nil
			//for _, drinker := range k.drinkers {
			//	select {
			//	case drinker.send <- message:
			//	default:
			//		close(drinker.send)
			//		delete(k.drinkers, drinker)
			//	}
			//}
		}
	}
}

func (k *Kettle) processDrinkOffer(drinkerId uuid.UUID) {
	drinker, ok := k.drinkers[drinkerId]
	if !ok {
		k.lgr.Warn("discarding offer-req for gone-away drinker", zap.String("drinkerId", drinkerId.String()))
		return
	}
	now := time.Now().UTC()
	if k.currentRoundTimestamp.IsZero() && (now.Sub(k.currentRoundTimestamp) < time.Minute*10) {
		// what happens here, app-user gets a chance to override and wipeout?
		drinker.sendResp(ErrorResponse("tea-round currently still in progress"))
	}
	k.currentMaker = drinker
	k.currentRoundTimestamp = time.Now().UTC()
	drinker.sendResp(SuccessResponse(""))
}

func (k *Kettle) processDrinkRequest(dr DrinkRequest) {
	msgBytes, err := dr.ToBytes()
	drinker, ok := k.drinkers[dr.DrinkerId]
	if !ok {
		k.lgr.Warn("discarding drink-req for gone-away drinker", zap.String("drinkerId", dr.DrinkerId.String()))
		return
	}

	if k.currentMaker == nil {
		drinker.sendResp(ErrorResponse("nobody currently offering to make drinks"))
		return
	}

	// send message to requester indicating error
	if err != nil {
		problem := "error encoding drink request"
		k.lgr.Error(problem, zap.String("drinkerId", dr.DrinkerId.String()), zap.ByteString("Request", dr.Request))
		drinker.sendResp(ErrorResponse(problem))
		return
	}
	k.currentMaker.send <- msgBytes
	drinker.sendResp(SuccessResponse(""))
}
