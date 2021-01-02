package app

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/ThePianoDentist/fancy-a-brew/app/middleware"
	ws "github.com/ThePianoDentist/fancy-a-brew/deprecatedws"

	"github.com/ThePianoDentist/fancy-a-brew/fcm_client"

	"github.com/ThePianoDentist/fancy-a-brew/app_context"
	_ "github.com/lib/pq"

	//_ "github.com/jackc/pgx/v4"

	handlers "github.com/ThePianoDentist/fancy-a-brew/app/handlers"

	"go.uber.org/zap"

	"github.com/gorilla/mux"
)

type App struct {
	Router *mux.Router
	DB     *sql.DB
	appCtx *app_context.AppContext
}

func NewApp(lgr *zap.Logger, user, password, dbname string) *App {
	connectionString := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, password, dbname)

	var err error
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}
	hub := ws.NewHub(lgr)

	fcmClient := fcm_client.NewFCMController(lgr)
	appCtx := &app_context.AppContext{Hub: hub, Lgr: lgr, DB: db, FcmController: fcmClient}

	router := mux.NewRouter()
	// db shouldnt be in both app and appctx. prob needs to stay in appctx as handlers need to access it
	app := &App{Router: router, appCtx: appCtx, DB: db}
	app.setupRouter()
	return app
}

func (a *App) Run(addr string) {
	// prob need smarter way of authing user/kettle.
	//a.Router.HandleFunc("/kettles/{kettleId}/{userId}/offer/", app.PostOffer).Methods(http.MethodPost)
	//a.Router.HandleFunc("/kettles/{kettleId}/{userId}/request/", app.PostDrinkRequest).Methods(http.MethodPost)
	// Need to auth to a kettle. (Is a webserver needed, or can peer-2-peea.Router. that sounds hard.)
	if err := http.ListenAndServe(addr, a.Router); err != nil {
		log.Fatal("error running server: ", zap.Error(err))
	}
}

func (a *App) setupRouter() {
	// handle preflight/CORS requests
	a.Router.Methods(http.MethodOptions).HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			return
		})
	a.Router.HandleFunc("/", handlers.IndexHandler)
	a.Router.HandleFunc("/health/", handlers.HealthHandler)
	//a.Router.HandleFunc("/ws/", handlers.WebsocketHandler(a.appCtx.Hub))
	//a.Router.HandleFunc("/ws/new/{kettleName}/{userName}", handlers.WebsocketHandlerNew(hub, lgr))
	//a.Router.HandleFunc("/ws/{kettleId}/{userName}", handlers.WebsocketHandler(hub))
	//a.Router.HandleFunc("/ws/new/{kettleName}/{userName}", handlers.WebsocketHandlerNew(hub, lgr))
	a.Router.HandleFunc("/users/{userId}/", handlers.GetUser).Methods(http.MethodGet)
	a.Router.Methods(http.MethodPost).Path("/users/").Handler(&app_context.CtxHandler{a.appCtx, handlers.PostUser})
	a.Router.HandleFunc("/kettles/{kettleId}/", handlers.GetKettle).Methods(http.MethodGet)
	// maybe should just be get with query params for location + radius....however that would mean it'd be cacheable.
	// and might miss new kettles added.
	a.Router.Methods(http.MethodPost).Path("/kettles/list/").Handler(&app_context.CtxHandler{a.appCtx, handlers.GetHotSteamyKettlesInYourArea})
	a.Router.Methods(http.MethodPost).Path("/kettles/").Handler(&app_context.CtxHandler{a.appCtx, handlers.PostKettle})
	a.Router.Methods(http.MethodPost).Path("/kettles/{kettleId}/offer/").Handler(&app_context.CtxHandler{a.appCtx, handlers.PostOfferBrew})
	a.Router.Methods(http.MethodPost).Path("/kettles/{kettleId}/response/").Handler(&app_context.CtxHandler{a.appCtx, handlers.PostBrewResponse})
	a.Router.Methods(http.MethodPost).Path("/kettles/{kettleId}/finished/").Handler(&app_context.CtxHandler{a.appCtx, handlers.PostFinished})
	a.Router.Use(middleware.AccessControl)
	a.Router.Use(middleware.RequireJsonContentType)
}
