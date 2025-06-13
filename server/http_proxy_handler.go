package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"proxy_server/log"
	"proxy_server/server/sniffing"
	"proxy_server/server/sniffing/tls"
)

func (m *manager) httpTcpConn(ctx context.Context, conn net.Conn, req *http.Request) {
	auth := req.Header.Get("Proxy-Authorization")
	auth = strings.Replace(auth, "Basic ", "", 1)
	authData, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		log.Error("[tcp_conn_handler] http代理Proxy-Authorization获取失败", zap.Error(err), zap.Any("auth", auth))
		if _, err = conn.Write([]byte("HTTP/1.1 407 Proxy Authorization Required\r\nProxy-Authenticate: Basic realm=\"Secure Proxys\"\r\n\r\n")); err != nil {
			return
		}
		return
	}

	proxyServerConn := conn.LocalAddr().(*net.TCPAddr)
	proxyServerIpStr := proxyServerConn.IP.String()
	peerIp := conn.RemoteAddr().(*net.TCPAddr).IP.String()

	userPasswdPair := strings.Split(string(authData), ":")
	if len(userPasswdPair) != 2 {
		log.Error("[tcp_conn_handler] http代理账号密码错误", zap.Any("authData", authData))
		if _, err = conn.Write([]byte("HTTP/1.1 407 Proxy Authorization Required\r\nProxy-Authenticate: Basic realm=\"Secure Proxys\"\r\n\r\n")); err != nil {
			return
		}
		return
	}

	proxyUserName := userPasswdPair[0]
	proxyPassword := userPasswdPair[1]
	exitIpStr := ""
	exitIpStr, err = m.auth.Valid(ctx, proxyUserName, proxyPassword, proxyServerIpStr, peerIp)
	if err != nil {
		log.Error("[tcp_conn_handler] http代理鉴权失败", zap.Error(err))
		if _, err = conn.Write([]byte("HTTP/1.1 407 Proxy Authorization Required\r\nProxy-Authenticate: Basic realm=\"Secure Proxys\"\r\n\r\n")); err != nil {
			return
		}
		return
	}

	exitIp := net.ParseIP(exitIpStr)

	targetHost := req.Host
	ipAndTarget := fmt.Sprintf("%s-%s", exitIpStr, targetHost)

	in := m.IpAndTargetIsInBlacklist(ipAndTarget)
	if in {
		msg := fmt.Sprintf("ip[%s]Andserver:[%s] in black list,unable to access!", exitIpStr, targetHost)
		log.Error("[tcp_conn_handler] " + msg)
		if _, err = conn.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\n")); err != nil {
			return
		}
		return
	}
	if ok, ipCount := m.AddIpConnCount(exitIpStr); ok {
		defer m.ReduceIpConnCount(exitIpStr)
	} else {
		///ip的连接数到达上限
		log.Error("[tcp_conn_handler] ip连接数到达上线", zap.Any("ip", exitIpStr), zap.Any("user", proxyUserName), zap.Any("连接数", ipCount))
		if _, err = conn.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\n")); err != nil {
			return
		}
		return

	}

	address := req.Host
	_, port, _ := net.SplitHostPort(req.Host)
	if req.Method == "CONNECT" {
		if port == "" {
			address = fmt.Sprint(req.Host, ":", 443)
		}
	} else {
		if port == "" {
			address = fmt.Sprint(req.Host, ":", 80)
		}
	}

	domain := regexpDomain(address)
	if domain != "" {
		if black, in := m.IsInBlacklist(domain); in {
			m.SendBlackListAccessLogMessageData(proxyUserName, proxyPassword, black, 1, proxyUserName, exitIpStr)
			log.Error("[tcp_conn_handler] 黑名单", zap.Any("domain", domain), zap.Any("local_ip", exitIpStr), zap.Any("target_addr", address), zap.Any("user", proxyUserName))
			if _, err = conn.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\n")); err != nil {
				return
			}
			return
		}
	}

	var domainPointer atomic.Pointer[string]
	domainPointer.Store(&domain)

	var target net.Conn
	target, err = DialContext(ctx, "tcp", address, time.Second*10, exitIp, 0)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			m.dialFailTracker.RecordDialFailConnection(ipAndTarget)
			log.Error("[tcp_conn_handler] tcp dial target timeout!", zap.Any("local_ip", exitIpStr), zap.Any("target_addr", address), zap.Any("user", proxyUserName))
		}
		log.Error("[tcp_conn_handler] 创建目标连接失败", zap.Any("local_ip", exitIpStr), zap.Any("target_addr", address), zap.Any("user", proxyUserName))
		if _, err = conn.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\n")); err != nil {
			return
		}
		return
	}
	defer target.Close()

	if req.Method == "CONNECT" {
		if _, err := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
			return
		}
	} else {
		req.Header.Del("Proxy-Authorization")
		var buf bytes.Buffer
		if err := req.Write(&buf); err != nil {
			log.Error("[tcp_conn_handler] 清除Proxy-Authorization", zap.Error(err))
			return
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			select {
			case <-ctx.Done():
				target.Close()
			case done <- struct{}{}:
			}
		}()

		if _, err := target.Write(buf.Bytes()); err != nil {
			for range done {
			}
			log.Error("[tcp_conn_handler] 转发http请求数据失败", zap.Error(err))
			return
		}
		for range done {
		}

	}

	// key := fmt.Sprintf("%s:%s", proxyUserName, proxyServerIpStr)
	key := proxyUserName
	connCtx := m.addUserConnection(key)
	action := connCtx.a
	defer m.deleteUserConnection(key, connCtx)

	var netConn, netTarget io.ReadWriteCloser

	netConn = newConn(conn, CONN_WRITE_TIME, CONN_READ_TIME)
	netTarget = newConn(target, CONN_WRITE_TIME, CONN_READ_TIME)

	byteChan := make(chan []byte, 1)
	defer close(byteChan)

	errCh := make(chan error, 2)
	defer close(errCh)

	done := make(chan struct{})
	wg := sync.WaitGroup{}
	defer func() {
		close(done)
		netConn.Close()
		netTarget.Close()
		wg.Wait()

		domain := domainPointer.Load()
		if domain != nil && *domain != "" {
			m.ReportAccessLogToInfluxDB(proxyUserName, *domain, exitIpStr)
		} else {
			hostArr := strings.Split(address, ":")
			if cap(hostArr) > 0 {
				m.ReportAccessLogToInfluxDB(proxyUserName, hostArr[0], exitIpStr)
			} else {
				m.ReportAccessLogToInfluxDB(proxyUserName, address, exitIpStr)
			}
		}
	}()

	///域名为空，并且使用CONNECT
	if domain == "" && req.Method == "CONNECT" {
		readWriterNotice, err := sniffing.NewReadWriterNotice(
			netConn,
			nil,
			func(buf []byte) {
				byteChan <- buf
			})
		if err != nil {
			return
		}
		netConn = readWriterNotice
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-done:
				return
			case buf, ok := <-byteChan:
				if len(buf) > 0 && ok {

					// 如果数据的第一个字节是 0x16，可能是 TLS 握手的 ClientHello 消息
					if buf[0] == 0x16 {
						// 创建一个 ClientHelloMsg 实例
						clientHelloMsg := tls.ClientHelloMsg{}
						// 尝试将负载数据反序列化为 ClientHelloMsg 实例
						clientHelloMsg.UnmarshalByByte(buf)
						// 如果反序列化后得到了 ServerName
						if clientHelloMsg.ServerName != "" {
							domainPointer.Store(&clientHelloMsg.ServerName)
							return
						}
					}

					// 解析 HTTP 请求
					hr, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(buf)))
					// 如果解析成功
					if err == nil {
						// 从 HTTP 请求头中获取 Host 字段作为 ServerName
						ServerName := hr.Header.Get("Host")
						if ServerName != "" {
							domainPointer.Store(&ServerName)
							return
						}

					}

					ServerName := regexpDomain(string(buf))

					if ServerName != "" {
						domainPointer.Store(&ServerName)
					}
				}
			}
		}()

	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := io.CopyBuffer(netTarget, NewLimitedReader(connCtx.ctx, netConn, action), make([]byte, 2*1024))
		errCh <- err
	}()

	go func() {
		defer wg.Done()
		_, err := io.CopyBuffer(netConn, NewLimitedReader(connCtx.ctx, netTarget, action), make([]byte, 2*1024))
		errCh <- err
	}()

	loopTime := 30 * time.Second
	ticker := time.NewTicker(loopTime)
	defer ticker.Stop()

	for {
		ticker.Reset(loopTime)
		select {

		case <-ticker.C:
			domain := domainPointer.Load()
			if domain != nil && *domain != "" {
				if black, in := m.IsInBlacklist(*domain); in {
					m.SendBlackListAccessLogMessageData(proxyUserName, proxyPassword, black, 1, proxyUserName, exitIpStr)
					log.Error("[tcp_conn_handler] 黑名单定时检测",
						zap.Any("domain", domain),
						zap.Error(err),
						zap.Any("username", proxyUserName),
						zap.Any("clientAddr", exitIpStr),
						zap.Any("target_host", address),
					)

					return
				}
			}
		case err, _ := <-errCh:
			if err != nil {
				log.Error("[tcp_conn_handler] conn close!",
					zap.Error(err),
					zap.Any("username", proxyUserName),
					zap.Any("clientAddr", exitIpStr),
					zap.Any("target_host", address),
				)
			}

			return
		case <-ctx.Done():
			return
		case <-connCtx.ctx.Done():
			return

		}
	}
}
