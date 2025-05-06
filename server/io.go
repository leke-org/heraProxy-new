package server

import (
	"context"
	"io"
	"net"
	"sync/atomic"
	"time"
)

const (
	HTTP_LOG_PRE              = "http_proxy:"
	SOCKS5_LOG_PRE            = "socks5_proxy:"
	LOG_PRE                   = "Title:"
	LOG_MESSAGE               = "msg:"
	PROTOCEL_TPC              = "tcp"
	PROTOCEL_UDP              = "udp"
	PROTOCEL_SOCKS5           = "socks5"
	PROTOCEL_HTTP             = "http"
	PROTOCEL_SPS              = "sps"
	ZERO_INT                  = 0
	REQ_METHOD_CONNECT        = "CONNECT"
	CONN_WRITE_TIME     int64 = 180
	CONN_READ_TIME      int64 = 180
	IN_STR                    = "in"
	OUT_STR                   = "out"
	EOF_STR                   = "EOF"
	DISCONNECT_CHANNEL        = "disconnect"
	REDIS_AUTH_USERDATA       = "auth_user_data"
	REDIS_USER_IPSET          = "user_ip_set"
)

type connContext struct {
	ctx    context.Context
	cancel context.CancelFunc
	a      *LimitedReaderAction
	c      uint64
}

type IpConnCountMapData struct {
	count atomic.Int64
}

func newConn(conn net.Conn, writetimeout, readtimeout int64) io.ReadWriteCloser {
	return &Conn{
		conn:         conn,
		readtimeout:  time.Duration(readtimeout) * time.Second,
		writetimeout: time.Duration(writetimeout) * time.Second,
	}
}

type Conn struct {
	conn         net.Conn
	readtimeout  time.Duration
	writetimeout time.Duration
}

func (c *Conn) Write(p []byte) (n int, err error) {
	// 设置写超时
	c.conn.SetWriteDeadline(time.Now().Add(c.writetimeout))
	return c.conn.Write(p)
}

func (c *Conn) Read(p []byte) (n int, err error) {
	c.conn.SetReadDeadline(time.Now().Add(c.readtimeout))
	return c.conn.Read(p)
}

func (c *Conn) Close() error {
	return c.conn.Close()
}
