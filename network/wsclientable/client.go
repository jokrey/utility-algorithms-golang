package wsclientable

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// This file gives us the Client Connection type. On a server it represents the connection to a client.
//   Internally it is a thread safe websocket connection, with an ID.
//   Certain wsclientable modules use that ID to distinguish connections and route messages.
// However it can also be used as a connection to the server on the client side.
//   For that, the 'Connect' constructor can be used.
//
// Apart from thread safety and ID, ClientConnections add only 1 important thing to websockets, Typed messages:
//   Typed Messages can be sent using 'SendTyped' and 'SendMapTyped'(for json support)
//   Typed Messages can be received over the 'ListenLoop', note that ListenLoop blocks and it can be advisable to run it in a goroutine
type ClientConnection struct {
	// stringified type but golang does not(yet) support generics, because 'we don't need it'
	//   (which is why they now add it to the language, just like Java did)
	//   (making the same mistake twice)
	ID       string
	raw      *websocket.Conn
	writeMut *sync.Mutex
}

func (c ClientConnection) SendRaw(text string) error {
	c.writeMut.Lock()
	defer c.writeMut.Unlock()

	return c.raw.WriteMessage(websocket.TextMessage, []byte(text))
}

// will marshal the given map into json and call SendTyped
func (c ClientConnection) SendMapTyped(mType string, data map[string]interface{}) error {
	byt, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("could not marshal json: %w", err)
	}

	return c.SendTyped(mType, string(byt))
}

func (c ClientConnection) SendTyped(mType string, data string) error {
	return c.SendRaw("{\"type\":\"" + mType + "\", \"data\":" + data + "}")
}

func (c ClientConnection) Close() error {
	c.writeMut.Lock()
	defer c.writeMut.Unlock()

	return c.raw.Close()
}

// Connect this websocket to the given url (example: http://dns.com:8080/route?user=testUserName)
// Server at url must be a wsclientable-server for reliable results
// On handshake problems, check cert (correct domain, still valid, added to local trusted)
func Connect(url string) (*ClientConnection, error) {
	// also works for https
	if strings.HasPrefix(url, "http") {
		url = "ws" + url[4:]
	}

	raw, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not dial to url(%v), error: %w", url, err)
	}

	return &ClientConnection{ID: "SERVER AT: " + url, raw: raw, writeMut: new(sync.Mutex)}, nil
}

const PingInterval = 66

type ClientCloseMessage struct {
	code int
	text string
}

// enables the listen loop, which will serve the given message handlers.
// will only return when this connection is closed, so it will typically be run in a goroutine
// Returns the close code and the closing message (1000 indicates normal closing)
func (c ClientConnection) ListenLoop(messageHandlers MessageHandlers) (int, string) {
	in := make(chan []byte)
	pingTicker := time.NewTicker(PingInterval * time.Second)
	stop := make(chan ClientCloseMessage)

	go func() {
		for {
			wsMessageType, message, err := c.raw.ReadMessage()
			if err != nil {
				if coc, ok := err.(*websocket.CloseError); ok {
					stop <- ClientCloseMessage{code: coc.Code, text: coc.Text}
				} else if coc, ok := err.(*net.OpError); ok {
					stop <- ClientCloseMessage{code: websocket.ClosePolicyViolation, text: coc.Error()}
				} else {
					stop <- ClientCloseMessage{code: 1000, text: err.Error()}
				}
				return
			}

			if wsMessageType != websocket.TextMessage {
				log.Println("Received message with unexpected type {}", wsMessageType)
			}
			in <- message
		}
	}()

	for {
		select {
		case <-pingTicker.C:
			if err := c.raw.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("Could not send ping - already closed?")
				_ = c.raw.Close()
				break //go back to select, expect message in close channel
			}
		case message := <-in:
			{
				var messageJSON map[string]interface{}
				if err := json.Unmarshal(message, &messageJSON); err != nil {
					log.Println("Received unparsable json from ", c.ID, ", closing connection")
					log.Println(string(message))
					_ = c.raw.Close()

					break //go back to select, expect message in close channel
				}
				mType := messageJSON["type"].(string)
				handler := messageHandlers[mType]
				if handler != nil {
					handler(mType, c, messageJSON["data"].(map[string]interface{}))
				} else {
					log.Println("Received unrecognised type ", mType, " from ", c.ID, ", closing connection")
					_ = c.raw.Close()
				}
			}
		case closeMessage := <-stop:
			return closeMessage.code, closeMessage.text
		}
	}
}
