package wsclientable

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

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

// Connect this websocket to the given url
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

// enables the listen loop, which will serve the given message handlers.
// will only return when this connection is closed, so it will typically be run in a goroutine
func (c ClientConnection) ListenLoop(
	messageHandlers map[string]func(string, ClientConnection, map[string]interface{}),
	connClosedHandlers []func(ClientConnection, int, string)) {
	in := make(chan []byte)
	stop := make(chan struct{})
	pingTicker := time.NewTicker(PingInterval * time.Second)

	go func() {
		for {
			wsMessageType, message, err := c.raw.ReadMessage()
			if err != nil {
				var coc *websocket.CloseError
				if k := errors.Is(err, coc); k {
					for _, connClosed := range connClosedHandlers {
						connClosed(c, coc.Code, coc.Text)
					}
				} else {
					var coc *net.OpError
					if k := errors.Is(err, coc); k {
						for _, connClosed := range connClosedHandlers {
							connClosed(c, 1008, coc.Error())
						}
					} else {
						for _, connClosed := range connClosedHandlers {
							connClosed(c, 1000, err.Error())
						}
					}
				}

				close(stop)
				break
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
				return
			}
		case message := <-in:
			{
				var messageJSON map[string]interface{}
				if err := json.Unmarshal(message, &messageJSON); err != nil {
					log.Println("Received unparsable json from ", c.ID, ", closing connection")
					log.Println(string(message))
					_ = c.raw.Close()

					return
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
		case <-stop:
			return
		}
	}
}
