package signaling

import (
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
	"gopkg.in/ini.v1"
	"log"
)

// minimal example config, with all required fields and some comments in example_configs/simplest_unencrypted.ini
//  Starts a room signaling server with all possible controllers (config see ini)
//  Allows direct forwarding to other KNOWN users (on message types offer, answer, candidate)
func StartSimplestSignalingServerUnencrypted(cfg *ini.File) {
	bindAddress := cfg.Section("signaling").Key("address").String()
	bindPort, _ := cfg.Section("signaling").Key("port").Int()
	httpRoute := cfg.Section("signaling").Key("http_route").String()

	base := wsclientable.NewWSHandlingServer()
	base.SetAuthenticator(wsclientable.AuthenticateUserPermitAll())
	base.AddDirectForwardingFunctionality("offer", "answer", "candidate")

	log.Printf("Started SignalingServer on %v:%v", bindAddress, bindPort)
	err := base.StartUnencrypted(bindAddress, bindPort, httpRoute)
	if err != nil {
		log.Fatal("Failed to start https server - with error: ", err)
	}
}

// minimal example config, with all required fields and some comments in example_configs/simplest_ssl.ini
//  Starts a room signaling server with all possible controllers (config see ini)
//  Allows direct forwarding to other KNOWN users (on message types offer, answer, candidate)
func StartSimplestSignalingServerWithSSL(cfg *ini.File) {
	bindAddress := cfg.Section("signaling").Key("address").String()
	bindPort, _ := cfg.Section("signaling").Key("port").Int()
	httpRoute := cfg.Section("signaling").Key("http_route").String()

	base := wsclientable.NewWSHandlingServer()
	base.SetAuthenticator(wsclientable.AuthenticateUserPermitAll())
	base.AddDirectForwardingFunctionality("offer", "answer", "candidate")

	log.Printf("Started SignalingServer on %v:%v", bindAddress, bindPort)
	err := base.StartWithTLSMultipleCerts(bindAddress, bindPort, httpRoute, wsclientable.ReadMultipleCertsFromCfg(cfg)...)
	if err != nil {
		log.Fatal("Failed to start https server - with error: ", err)
	}
}
