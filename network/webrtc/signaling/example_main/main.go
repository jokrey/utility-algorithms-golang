package main

import (
	"github.com/jokrey/utility-algorithms-golang/network/webrtc/signaling"
	"gopkg.in/ini.v1"
	"log"
)

func main() {
	cfg, err := ini.Load("configs/config.ini")
	if err != nil {
		log.Fatalf("Fail to read file: %v", err)
	}
	// signaling.StartSimplestSignalingServerWithSSL(cfg)
	signaling.StartRoomSignalingServerWithSSL(cfg)
}