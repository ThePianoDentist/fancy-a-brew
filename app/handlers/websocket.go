package app

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/gorilla/mux"

	"github.com/ThePianoDentist/fancy-a-brew/ws"
)

func WebsocketHandler(hub *ws.Hub) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		kettleId, err := uuid.FromBytes([]byte(vars["kettleId"]))
		userName := vars["userName"]
		if err != nil {
			http.Error(w, fmt.Sprintf("expected uuid kettleId. Got: %s", vars["kettleId"]), http.StatusBadRequest)
		}
		kettle, ok := hub.GetKettle(kettleId)
		if !ok {
			http.NotFound(w, r)
			return
		}
		ws.ServeWs(kettle, userName, w, r)
	}
}

//func WebsocketHandler(hub *ws.Hub) func(w http.ResponseWriter, r *http.Request) {
//	return func(w http.ResponseWriter, r *http.Request) {
//		vars := mux.Vars(r)
//		kettleId, err := uuid.FromBytes([]byte(vars["kettleId"]))
//		userName := vars["userName"]
//		if err != nil {
//			http.Error(w, fmt.Sprintf("expected uuid kettleId. Got: %s", vars["kettleId"]), http.StatusBadRequest)
//		}
//		kettle, ok := hub.GetKettle(kettleId)
//		if !ok {
//			http.NotFound(w, r)
//			return
//		}
//		ws.ServeWs(kettle, userName, w, r)
//	}
//}
//
//func WebsocketHandlerNew(hub *ws.Hub, lgr *zap.Logger) func(w http.ResponseWriter, r *http.Request) {
//	return func(w http.ResponseWriter, r *http.Request) {
//		vars := mux.Vars(r)
//		kettleName := vars["kettleName"]
//		userName := vars["userName"]
//		newKettle := hub.AddKettle(lgr, kettleName)
//		ws.ServeWs(newKettle, userName, w, r)
//	}
//}
