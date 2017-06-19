package network

import (
	"net"
	"time"
	"strconv"
)

type MCNP_Client struct {
	MCNP_Connection
}

func New_MCNP_Client_Connection(server_ip_or_dns_url string, port int, timeout time.Duration) (MCNP_Client, error) {
	conn, err := net.DialTimeout("tcp", server_ip_or_dns_url+":"+strconv.Itoa(port), timeout)

	return MCNP_Client{New_MCNP_Connection(conn)}, err
}