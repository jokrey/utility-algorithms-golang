package wsclientable_test

import (
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
	"testing"
	"time"
)

func TestConnectAndSimpleBackAndForth(t *testing.T) {
	go func() {
		time.Sleep(1 * time.Second)
		client, err := wsclientable.Connect("http://localhost:7011/testHello?user=test")
		if err != nil {
			t.Fatal(err)
		}
		clientMessageHandlers := make(map[string]func(string, wsclientable.ClientConnection, map[string]interface{}))
		clientMessageHandlers["hello"] = func(_ string, _ wsclientable.ClientConnection, data map[string]interface{}) {
			println("Client received reply: ", data["reply"].(string))
			_ = client.Close()
		}

		err = client.SendTyped("hello", "{\"msg\":\"i am a client\"}")
		if err != nil {
			t.Fatal("Error - Consider test failed")
		}

		client.ListenLoop(clientMessageHandlers)
	}()

	server := wsclientable.NewWSHandlingServer()
	server.SetAuthenticator(wsclientable.AuthenticateUserPermitAll())

	closed := false

	server.AddMessageHandler("hello", func(mType string, client wsclientable.ClientConnection, data map[string]interface{}) {
		println("Server received message(from: ", client.ID, "): ", data["msg"].(string))
		err := client.SendTyped("hello", "{\"reply\":\"you said: "+data["msg"].(string)+"\"}")
		if err != nil {
			t.Fatal(err)
		}
		err = client.Close()
		if err != nil {
			t.Fatal(err)
		}

		closed = true
		_ = server.Close()
	})

	println("Started test server")
	err := server.StartUnencrypted("localhost", 7011, "/testHello")
	if err != nil && !closed {
		t.Fatalf("Error - Consider test failed, error: %v", err)
	}
}
