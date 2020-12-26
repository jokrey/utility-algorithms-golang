package signaling

import (
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
	"gopkg.in/ini.v1"
	"log"
)

// minimal example config, with all required fields and some comments in example_configs/room_unencrypted.ini
//  Starts a room signaling server with all possible controllers (config see ini)
//  Within rooms, it will allow direct forwarding to other KNOWN users (on message types offer, answer, candidate)
func StartRoomSignalingServerUnencrypted(cfg *ini.File) {
	bindAddress := cfg.Section("bind").Key("address").String()
	bindPort, _ := cfg.Section("bind").Key("port").Int()
	httpRoute := cfg.Section("signaling").Key("http_route").String()

	base := wsclientable.NewWSHandlingServer()
	base.AddRoomForwardingFunctionality(wsclientable.BundleControllers(
		wsclientable.NewPermanentRoomControllerFromCFG(cfg),
		wsclientable.NewHTTPEditableRoomControllerFromCFG(cfg),
		wsclientable.NewTemporaryRoomControllerFromCFG(cfg),
	), "offer", "answer", "candidate")

	log.Printf("Started RoomSignalingServer on %v:%v", bindAddress, bindPort)
	err := base.StartUnencrypted(bindAddress, bindPort, httpRoute)
	if err != nil {
		log.Fatal("Failed to start https server - with error: ", err)
	}
}

// minimal example config, with all required fields and some comments in example_configs/room_ssl.ini
//  Starts a room signaling server with all possible controllers (config see ini)
//  Within rooms, it will allow direct forwarding to other KNOWN users (on message types offer, answer, candidate)
func StartRoomSignalingServerWithSSL(cfg *ini.File) {
	bindAddress := cfg.Section("bind").Key("address").String()
	bindPort, _ := cfg.Section("bind").Key("port").Int()
	httpRoute := cfg.Section("signaling").Key("http_route").String()

	base := wsclientable.NewWSHandlingServer()
	base.AddRoomForwardingFunctionality(wsclientable.BundleControllers(
		wsclientable.NewPermanentRoomControllerFromCFG(cfg),
		wsclientable.NewHTTPEditableRoomControllerFromCFG(cfg),
		wsclientable.NewTemporaryRoomControllerFromCFG(cfg),
	), "offer", "answer", "candidate")

	log.Printf("Started RoomSignalingServer on %v:%v", bindAddress, bindPort)
	err := base.StartWithTLSMultipleCerts(bindAddress, bindPort, httpRoute, wsclientable.ReadMultipleCertsFromCfg(cfg)...)
	if err != nil {
		log.Fatal("Failed to start https server - with error: ", err)
	}
}
