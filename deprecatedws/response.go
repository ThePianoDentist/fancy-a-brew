package ws

type Response struct {
	Status string `json:"status"`
	Msg    string `json:"msg"`
}

func ErrorResponse(msg string) Response {
	return Response{Status: "error", Msg: msg}
}

func SuccessResponse(msg string) Response {
	return Response{Status: "ok", Msg: msg}
}
