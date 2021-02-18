package signaling

import (
	"gopkg.in/ini.v1"
	"log"
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
)

// minimal example config, with all required fields and some comments in example_configs/room_unencrypted.ini
//  Starts a room signaling server with all possible controllers (config see ini)
//  Within rooms, it will allow direct forwarding to other KNOWN users (on message types offer, answer, candidate)
func StartRoomSignalingServerUnencrypted(cfg *ini.File) {
	bindAddress := cfg.Section("signaling").Key("address").String()
	bindPort, _ := cfg.Section("signaling").Key("port").Int()
	httpRoute := cfg.Section("signaling").Key("http_route").String()

	base := wsclientable.NewWSHandlingServer()
	base.AddRoomForwardingFunctionality(wsclientable.BundleControllers(
		wsclientable.NewPermanentRoomControllerFromCFG(cfg),
		wsclientable.NewHTTPRoomEditorFromCFG(cfg),
		wsclientable.NewHTTPTemporaryRoomEditorFromCFG(cfg),
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
	bindAddress := cfg.Section("signaling").Key("address").String()
	bindPort, _ := cfg.Section("signaling").Key("port").Int()
	httpRoute := cfg.Section("signaling").Key("http_route").String()

	base := wsclientable.NewWSHandlingServer()
	base.AddRoomForwardingFunctionality(wsclientable.BundleControllers(
		wsclientable.NewPermanentRoomControllerFromCFG(cfg),
		wsclientable.NewHTTPRoomEditorFromCFG(cfg),
		wsclientable.NewHTTPTemporaryRoomEditorFromCFG(cfg),
	), "offer", "answer", "candidate")

	log.Printf("Started RoomSignalingServer on %v:%v", bindAddress, bindPort)
	err := base.StartWithTLSMultipleCerts(bindAddress, bindPort, httpRoute, wsclientable.ReadMultipleCertsFromCfg(cfg)...)
	if err != nil {
		log.Fatal("Failed to start https server - with error: ", err)
	}
}

//  Starts a room signaling server with only the given room controllers
//  Within rooms, it will allow direct forwarding to other KNOWN users (on message types offer, answer, candidate)
func StartRoomSignalingServerWithSSLWithCustomControllers(cfg *ini.File, controllers ...wsclientable.RoomControllerI) {
	bindAddress := cfg.Section("signaling").Key("address").String()
	bindPort, _ := cfg.Section("signaling").Key("port").Int()
	httpRoute := cfg.Section("signaling").Key("http_route").String()

	base := wsclientable.NewWSHandlingServer()
	base.AddRoomForwardingFunctionality(wsclientable.BundleControllers(controllers...),
		"offer", "answer", "candidate")

	log.Printf("Started RoomSignalingServer on %v:%v", bindAddress, bindPort)
	err := base.StartWithTLSMultipleCerts(bindAddress, bindPort, httpRoute, wsclientable.ReadMultipleCertsFromCfg(cfg)...)
	if err != nil {
		log.Fatal("Failed to start https server - with error: ", err)
	}
}

//  Starts a room signaling server with only the given room controllers
//  Within rooms, it will allow direct forwarding to other KNOWN users (on message types offer, answer, candidate)
func StartRoomSignalingServerUnencryptedWithCustomControllers(cfg *ini.File, controllers ...wsclientable.RoomControllerI) {
	bindAddress := cfg.Section("signaling").Key("address").String()
	bindPort, _ := cfg.Section("signaling").Key("port").Int()
	httpRoute := cfg.Section("signaling").Key("http_route").String()

	base := wsclientable.NewWSHandlingServer()
	base.AddRoomForwardingFunctionality(wsclientable.BundleControllers(controllers...),
		"offer", "answer", "candidate")

	log.Printf("Started RoomSignalingServer on %v:%v", bindAddress, bindPort)
	err := base.StartUnencrypted(bindAddress, bindPort, httpRoute)
	if err != nil {
		log.Fatal("Failed to start https server - with error: ", err)
	}
}
