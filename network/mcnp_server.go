package network

import (
	"fmt"
	"net"
	"strconv"
)

type MCNP_Server struct {
	running bool
	port int
	handleConnection func(conn MCNP_Connection)
	port_listener net.Listener
}

//Golang Constructor
func New_MCNP_Server(port int, connectionHandler func(conn MCNP_Connection)) MCNP_Server {
	new := MCNP_Server{true, port, connectionHandler, nil}
	return new
}

func (server *MCNP_Server) RunListenerLoop() {
	tmp, err := net.Listen("tcp", ":"+strconv.Itoa(server.port))
	server.port_listener=tmp
	fmt.Println("RunListenerLoop: ", server.port_listener)
	if err != nil {
		fmt.Println("Starting server FAILED.. Sorry")
	} else {
		fmt.Println("\n=======! Server Started !=======\n")
		for server.running { //ever
			fmt.Println("=======! Listening to port", server.port, "for new connection !=======")
			conn, err := server.port_listener.Accept()
			if err != nil {
				fmt.Println("Sorry. There was an attempt at a connection, but it failed")
			} else {
				go server.handleConnection(New_MCNP_Connection(conn)) //in new goroutine
			}
		}
	}
}

func (server MCNP_Server) Close() {
	server.running = false
	fmt.Println("Close: ", server.port_listener)
	server.port_listener.Close()
}