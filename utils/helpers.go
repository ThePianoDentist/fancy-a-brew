package utils

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

func ErrorResp(lgr *zap.Logger, w http.ResponseWriter, code int, message string, err error) {
	if code >= 500 {
		lgr.Error("error processing request", zap.Error(err))
	} else {
		lgr.Warn("user error processing request", zap.Error(err))
	}
	JSONResponse(lgr, w, code, map[string]string{"status": "error", "error": message})
}

func SuccessResp(lgr *zap.Logger, w http.ResponseWriter, code int, data interface{}) {
	JSONResponse(lgr, w, code, map[string]interface{}{"status": "ok", "data": data})
}

func JSONResponse(lgr *zap.Logger, w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	// could return errors instead. Lift them up to calling funcs.
	// DOn't think it's a big deal for this scenario as writing out the JSON `should` be final action/logic of request.
	if err != nil {
		ErrorResp(lgr, w, code, "unexpected error writing json", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(response)
	if err != nil {
		lgr.Error("error writing response:", zap.Error(err))
	}
}
