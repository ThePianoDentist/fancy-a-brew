package app

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/ThePianoDentist/fancy-a-brew/storage"
	"github.com/ThePianoDentist/fancy-a-brew/utils"

	"github.com/ThePianoDentist/fancy-a-brew/app_context"
)

func GetKettle(w http.ResponseWriter, r *http.Request) {
	return
}

type GetKettlesReq struct {
	MetreRadius int32
	Long        float64
	Lat         float64
}

type PostOfferBrewReq struct {
	FirebaseToken string
}

type PostBrewRespReq struct {
	FirebaseToken  string
	TheUsualTicked bool
	Choice         string
	Name           string
}

func GetHotSteamyKettlesInYourArea(appCtx *app_context.AppContext, w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var d GetKettlesReq
	if err := decoder.Decode(&d); err != nil {
		utils.ErrorResp(appCtx.Lgr, w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}
	kettles, err := storage.GetKettlesWithinRadius(appCtx.DB, d.Long, d.Lat, d.MetreRadius)
	if err != nil {
		utils.ErrorResp(appCtx.Lgr, w, 500, "unexpected goof getting kettles", err)
		return
	}
	utils.SuccessResp(appCtx.Lgr, w, 200, kettles)
	return
}

func PostKettle(appCtx *app_context.AppContext, w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var k storage.Kettle
	if err := decoder.Decode(&k); err != nil {
		utils.ErrorResp(appCtx.Lgr, w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}
	// I think reading body is weird/dumb. and defering before reading body leads to panic in some scenarios.
	// (add stack overflow link here if find/know)
	defer r.Body.Close()
	kettleId, err := k.UpsertKettle(appCtx.DB)
	if err != nil {
		utils.ErrorResp(appCtx.Lgr, w, http.StatusInternalServerError, "Fudge! Something went wrong. Bug reports to jkthepianodentist@gmail.com", err)
		return
	}
	utils.SuccessResp(appCtx.Lgr, w, 201, map[string]string{"kettleId": kettleId.String(), "name": k.Name})
}

func PostOfferBrew(appCtx *app_context.AppContext, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	kettleId, err := uuid.Parse(vars["kettleId"])
	if err != nil {
		utils.ErrorResp(appCtx.Lgr, w, http.StatusBadRequest, fmt.Sprintf("expected uuid kettleId. Got: %s", vars["kettleId"]), err)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var d PostOfferBrewReq
	if err := decoder.Decode(&d); err != nil {
		utils.ErrorResp(appCtx.Lgr, w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	userId, err := storage.GetUserIdFromToken(appCtx.DB, d.FirebaseToken)
	if err != nil {
		utils.ErrorResp(appCtx.Lgr, w, http.StatusInternalServerError, "Fudge! Something went wrong. Bug reports to jkthepianodentist@gmail.com", err)
		return
	}

	kettle, err := storage.GetKettle(appCtx.DB, kettleId)
	if err != nil {
		utils.ErrorResp(appCtx.Lgr, w, http.StatusInternalServerError, "Fudge! Something went wrong. Bug reports to jkthepianodentist@gmail.com", err)
		return
	}
	usersInRadius, err := storage.GetUsersWithinRadius(appCtx.DB, kettle.Long, kettle.Lat, 100)
	data := map[string]string{
		"kettleId":   kettleId.String(),
		"kettleName": kettle.Name,
		"type":       "offer",
		// TODO get username of maker (send firebase token and then do a user-lookup)
	}
	if err := storage.SetCurrentMaker(appCtx.DB, kettle.KettleId, userId); err != nil {
		utils.ErrorResp(appCtx.Lgr, w, http.StatusInternalServerError, "Fudge! Something went wrong. Bug reports to jkthepianodentist@gmail.com", err)
		return
	}
	for _, user := range usersInRadius {
		err := appCtx.FcmController.SendFCM(user.FirebaseToken, data)
		if err != nil {
			appCtx.Lgr.Error("error publishing fcm message", zap.Error(err))
			// Keep going as one failure shouldnt kill everything.
			// however might need to keep track of this when it comes to checking responses.
			// I guess as we don't wait for everyone to
		}
	}
	utils.SuccessResp(appCtx.Lgr, w, 200, struct{}{})
}

func PostBrewResponse(appCtx *app_context.AppContext, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	kettleId, err := uuid.Parse(vars["kettleId"])
	if err != nil {
		utils.ErrorResp(appCtx.Lgr, w, http.StatusBadRequest, fmt.Sprintf("expected uuid kettleId. Got: %s", vars["kettleId"]), err)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var d PostBrewRespReq
	if err := decoder.Decode(&d); err != nil {
		utils.ErrorResp(appCtx.Lgr, w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	kettle, err := storage.GetKettle(appCtx.DB, kettleId)
	if err != nil {
		utils.ErrorResp(appCtx.Lgr, w, http.StatusInternalServerError, "Fudge! Something went wrong. Bug reports to jkthepianodentist@gmail.com", err)
		return
	}
	if (kettle.CurrentMaker == uuid.UUID{}) {
		// TODO send message back saying too-late baby
		return
	}
	maker, err := storage.GetUser(appCtx.DB, kettle.CurrentMaker)
	if err != nil {
		utils.ErrorResp(appCtx.Lgr, w, http.StatusInternalServerError, "Fudge! Something went wrong. Bug reports to jkthepianodentist@gmail.com", err)
		return
	}
	if err := appCtx.FcmController.SendFCM(maker.FirebaseToken, map[string]string{"choice": d.Choice, "name": d.Name, "type": "drinkrequest"}); err != nil {
		appCtx.Lgr.Error("error publishing fcm message", zap.Error(err))
	}
	utils.SuccessResp(appCtx.Lgr, w, http.StatusOK, struct{}{})
}

func PostFinished(appCtx *app_context.AppContext, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	kettleId, err := uuid.Parse(vars["kettleId"])
	if err != nil {
		utils.ErrorResp(appCtx.Lgr, w, http.StatusBadRequest, fmt.Sprintf("expected uuid kettleId. Got: %s", vars["kettleId"]), err)
		return
	}
	if err := storage.SetCurrentMaker(appCtx.DB, kettleId, uuid.UUID{}); err != nil {
		utils.ErrorResp(appCtx.Lgr, w, http.StatusInternalServerError, "Fudge! Something went wrong. Bug reports to jkthepianodentist@gmail.com", err)
		return
	}
	utils.SuccessResp(appCtx.Lgr, w, http.StatusOK, struct{}{})
}
