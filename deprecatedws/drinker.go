package ws

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/google/uuid"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Drinker is a middleman between the websocket connection and the kettle.
type Drinker struct {
	Id   uuid.UUID
	name string
	lgr  *zap.Logger
	// I initially just passed the kettle-chans into drinker, for it to send through them.
	// Because to me kettle owning map of drinkers, and then drinker owning reference to kettle is smelly code.
	// But that's what we really have. why fight it :)
	kettle *Kettle

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

func NewDrinker(lgr *zap.Logger, kettle *Kettle, name string, conn *websocket.Conn) *Drinker {
	// TODO 256. do i need to overflow protect? or will it error anyway?
	return &Drinker{Id: uuid.New(), lgr: lgr, kettle: kettle, name: name, conn: conn, send: make(chan []byte, 256)}
}

// readPump pumps messages from the websocket connection to the kettle.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (d *Drinker) readPump() {
	defer func() {
		d.kettle.unregister <- d.Id
		d.conn.Close()
	}()
	d.conn.SetReadLimit(maxMessageSize)
	d.conn.SetReadDeadline(time.Now().Add(pongWait))
	d.conn.SetPongHandler(func(string) error { d.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := d.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		var request Request
		if err := json.Unmarshal(message, &request); err != nil {
			// log stuff
			log.Printf("error: %v", err)
			d.sendResp(ErrorResponse("Could not parse request"))
		}
		switch request.Method {
		case "offer":
			d.kettle.drinkOffers <- d.Id
		case "request":
			d.kettle.drinkRequests <- DrinkRequest{DrinkerId: d.Id, DrinkerName: d.name, Request: message}
		case "completion":
			d.kettle.roundCompleted <- struct{}{}
		default:
			d.lgr.Warn("unexpected request method", zap.String("Method", request.Method))
		}
	}
}

// writePump pumps messages from the kettle to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (d *Drinker) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		d.conn.Close()
	}()
	for {
		select {
		case message, ok := <-d.send:
			d.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The kettle closed the channel.
				d.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := d.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(d.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-d.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			d.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := d.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (d *Drinker) sendResp(resp Response) {
	msgBytes, err := json.Marshal(resp)
	if err != nil {
		d.lgr.Error("could not marshall resp into bytes", zap.String("Msg", resp.Msg))
	}
	d.send <- msgBytes
}

// serveWs handles websocket requests from the peer.
func ServeWs(kettle *Kettle, username string, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	// TODO move these into readPump methods.
	// method to make a new kettle. and connect to existing kettle (as user, user-id and/or name?)
	drinker := NewDrinker(kettle.lgr, kettle, username, conn)
	kettle.register <- drinker

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go drinker.writePump()
	go drinker.readPump()
}
