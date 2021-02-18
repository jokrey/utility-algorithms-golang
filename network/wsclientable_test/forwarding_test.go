package wsclientable_test

import (
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
	"testing"
	"time"
)

func TestForwarding(t *testing.T) {
	go func() {
		time.Sleep(1 * time.Second)
		client, err := wsclientable.ConnectAs("http://localhost:8011/testHello", "u1")
		if err != nil {
			t.Fatal(err)
		}
		clientMessageHandlers := make(map[string]func(string, wsclientable.ClientConnection, map[string]interface{}))
		clientMessageHandlers["close_remote_client"] =
			func(_ string, _ wsclientable.ClientConnection, data map[string]interface{}) {
				println("Client received forwarded - from: ", data["from"].(string))

				err = client.SendTyped("close_server", "{}")
				if err != nil {
					t.Fatal("Error - Consider test failed")
				}
				_ = client.Close()
			}

		client.ListenLoop(clientMessageHandlers)
	}()
	go func() {
		time.Sleep(2 * time.Second)
		client, err := wsclientable.Connect("http://localhost:8011/testHello?user=u2")
		if err != nil {
			t.Fatal(err)
		}

		err = client.SendTyped("close_remote_client", "{\"to\":\"u1\"}")
		if err != nil {
			t.Fatal("Error - Consider test failed")
		}

		_ = client.Close()
	}()

	server := wsclientable.NewWSHandlingServer()
	server.SetAuthenticator(wsclientable.AuthenticateUserPermitAll())

	closed := false

	server.AddDirectForwardingFunctionality("close_remote_client")

	server.AddMessageHandler("close_server",
		func(mType string, client wsclientable.ClientConnection, data map[string]interface{}) {
			closed = true
			_ = server.Close()
		},
	)

	println("Started test server")
	err := server.StartUnencrypted("localhost", 8011, "/testHello")
	if err != nil && !closed {
		t.Fatalf("Error - Consider test failed, error: %v", err)
	}
}
