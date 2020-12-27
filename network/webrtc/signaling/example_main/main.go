package main

import (
	"github.com/jokrey/utility-algorithms-golang/network/webrtc/signaling"
	"gopkg.in/ini.v1"
	"log"
)

func main() {
	cfg, err := ini.Load("network/webrtc/signaling/example_configs/room_ssl.ini")
	if err != nil {
		log.Fatalf("Fail to read file: %v", err)
	}
	// signaling.StartSimplestSignalingServerWithSSL(cfg)
	signaling.StartRoomSignalingServerWithSSL(cfg)
}