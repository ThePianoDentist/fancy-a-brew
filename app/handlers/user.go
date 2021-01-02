package app

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/ThePianoDentist/fancy-a-brew/utils"

	"github.com/ThePianoDentist/fancy-a-brew/app_context"

	"github.com/ThePianoDentist/fancy-a-brew/storage"
)

func GetUser(w http.ResponseWriter, r *http.Request) {
	return
}

func PostUser(appCtx *app_context.AppContext, w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var u storage.User
	if err := decoder.Decode(&u); err != nil {
		utils.ErrorResp(appCtx.Lgr, w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}
	// I think reading body is weird/dumb. and defering before reading body leads to panic in some scenarios.
	// (add stack overflow link here if find/know)
	defer r.Body.Close()
	userId, err := u.UpsertUser(appCtx.DB)
	if err != nil {
		appCtx.Lgr.Error("error inserting user:", zap.Error(err))
		utils.ErrorResp(appCtx.Lgr, w, http.StatusInternalServerError, "Fudge! Something went wrong. Bug reports to jkthepianodentist@gmail.com", err)
		return
	}
	utils.SuccessResp(appCtx.Lgr, w, 201, map[string]string{"userId": userId.String()})
}
