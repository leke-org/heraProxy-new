package server

import (
	"context"
	"net"
	"time"
)

func DialContext(ctx context.Context, network, address string, timeout time.Duration, localIP []byte, localPort int) (net.Conn, error) {
	netAddr := &net.TCPAddr{Port: localPort}
	if len(localIP) != 0 {
		netAddr.IP = localIP
	}

	d := net.Dialer{Timeout: timeout, LocalAddr: netAddr}
	conn, err := d.DialContext(ctx, network, address)

	return conn, err
}
