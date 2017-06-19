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

///can very well be run in a different go routine.
///Then it is possible to Close it from the outside.
///Mostly however it makes sense to run it on the main thread. Since most server only want to handleConnections.
func (server *MCNP_Server) RunListenerLoop() error {
	tmp, err := net.Listen("tcp", ":"+strconv.Itoa(server.port))
	server.port_listener=tmp
	fmt.Println("RunListenerLoop: ", server.port_listener)
	if err == nil {
		fmt.Println("\n=======! Server Started !=======\n")
		for server.running {
			fmt.Println("=======! Listening to port", server.port, "for new connection !=======")
			conn, listenerr := server.port_listener.Accept()
			if listenerr != nil {
				fmt.Println("Sorry. There was an attempt at a connection, but it failed")
			} else {
				go func () {
					mcnp_conn := New_MCNP_Connection(conn);
					defer mcnp_conn.Close()
					server.handleConnection(mcnp_conn)
				}() //in new goroutine
			}
		}
	}
	return err
}

func (server MCNP_Server) Close() {
	server.running = false
	fmt.Println("Close: ", server.port_listener)
	server.port_listener.Close()
}