package wsclientable

import "log"

// Will add direct relay functionality to the server on the given message types.
//   for that it will keep a map of currently open connections
//   In the data field it will require a 'to' field,
//     that indicates which connection (with the given name) the message shall be forwarded to
//     the message will be forwarded as is with the original type and the data field exactly as is
//     only a 'from' field will be added/overridden - this from field is verified.
func (s *Server) AddDirectForwardingFunctionality(messageTypes ...string) {
	knownPeers := make(map[string]*ClientConnection)

	s.AddConnOpenedHandler(func(connection ClientConnection) {
		log.Printf("Connected: %v", connection.ID)
		// not thread safe:
		if _, exists := knownPeers[connection.ID]; !exists {
			knownPeers[connection.ID] = &connection
		}
	})
	s.AddConnClosedHandler(func(connection ClientConnection, code int, text string) {
		log.Printf("Disconnected: %v", connection.ID)
		delete(knownPeers, connection.ID)
	})

	directRelayHandler := func(mType string, client ClientConnection, data map[string]interface{}) {
		if to, ok := data["to"]; !ok || to == nil {
			log.Printf("Missing field for message of type %v", mType)

			return
		}

		to := data["to"].(string)
		data["from"] = client.ID

		peer, ok := knownPeers[to]
		if !ok {
			err := client.SendTyped(
				"error", "{\"requestType\":\""+mType+"\", \"reason\":\"Peer "+to+" not found\"}")
			if err != nil {
				log.Printf("Error sending to %v\n", client)
			}

			return
		}

		// relay to other client
		err := peer.SendMapTyped(mType, data)
		if err != nil {
			log.Printf("Error sending to %v", client)
		}
	}

	for _, mType := range messageTypes {
		s.AddMessageHandler(mType, directRelayHandler)
	}
}
