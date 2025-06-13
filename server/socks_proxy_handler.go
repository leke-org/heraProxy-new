package server

import (
	"bufio"
	"bytes"
	"context"
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
	"proxy_server/utils/socks5"
)

func (m *manager) socksTcpConn(ctx context.Context, conn net.Conn) {
	// 读取账号密码
	// var user, pwd string
	user, pwd, err := socks5.GetUserPassword(conn)
	if err != nil {
		log.Error("[socks_proxy_handler] 读取账号密码错误", zap.Error(err))
		if _, err = conn.Write([]byte{socks5.UserAuthVersion, socks5.AuthFailure}); err != nil {
			return
		}
		return
	}

	proxyServerConn := conn.LocalAddr().(*net.TCPAddr)
	proxyServerIpStr := proxyServerConn.IP.String()
	//proxyServerIpByte := proxyServerConn.IP.To4()
	peerIp := conn.RemoteAddr().(*net.TCPAddr).IP.String()
	isIpv4 := false
	exitIpStr := ""

	exitIpStr, err = m.auth.Valid(ctx, user, pwd, proxyServerIpStr, peerIp)
	if err != nil {
		log.Error("[socks_proxy_handler] 鉴权失败", zap.Error(err), zap.Any("user", user), zap.Any("pwd", pwd), zap.Any("ip", exitIpStr))
		if _, err = conn.Write([]byte{socks5.UserAuthVersion, socks5.AuthFailure}); err != nil {
			return
		}
		return
	}

	exitIp := net.ParseIP(exitIpStr)
	if exitIp.To4() != nil {
		isIpv4 = true
	}

	if ok, ipCount := m.AddIpConnCount(exitIpStr); ok {
		defer m.ReduceIpConnCount(exitIpStr)
	} else {
		// ip的连接数到达上限
		log.Error("[socks_proxy_handler] ip连接数达到上限", zap.Any("ip", exitIpStr), zap.Any("连接数", ipCount), zap.Any("user", user))
		resp := socks5.ConnectionRefused
		if err = socks5.SendReply(conn, resp, nil); err != nil {
			return
		}
		return

	}

	///认证成功，返回消息给客户端
	if _, err = conn.Write([]byte{socks5.UserAuthVersion, socks5.AuthSuccess}); err != nil {
		log.Error("[socks_proxy_handler] 认证成功，返回消息给客户端失败", zap.Any("ip", exitIpStr), zap.Any("user", user))
		return
	}

	var destAddr *socks5.AddrSpec
	destAddr, err = socks5.ReadDestAddr(conn)
	if err != nil {
		if err == socks5.UnrecognizedAddrType {
			if err = socks5.SendReply(conn, socks5.AddrTypeNotSupported, nil); err != nil {
				return
			}
		}
		return
	}

	ipAndTarget := fmt.Sprintf("%s-%s", exitIpStr, destAddr.String())
	in := m.IpAndTargetIsInBlacklist(ipAndTarget)
	if in {
		msg := fmt.Sprintf("ip[%s]Andserver:[%s] in black list,unable to access!", exitIpStr, destAddr.String())
		log.Error("[socks_proxy_handler] " + msg)
		if err = socks5.SendReply(conn, socks5.NetworkUnreachable, nil); err != nil {
			return
		}
		return
	}

	domain := regexpDomain(destAddr.Address())
	if domain != "" {
		if black, in := m.IsInBlacklist(domain); in {
			m.SendBlackListAccessLogMessageData(user, pwd, black, 1, user, exitIpStr)
			log.Error("[socks_proxy_handler] 黑名单", zap.Any("domain", domain), zap.Any("local_ip", exitIpStr), zap.Any("target_addr", destAddr.Address()), zap.Any("user", user))
			if err = socks5.SendReply(conn, socks5.HostUnreachable, nil); err != nil {
				return
			}
			return
		}
	}

	var domainPointer atomic.Pointer[string]
	domainPointer.Store(&domain)

	var target net.Conn
	target, err = DialContext(ctx, "tcp", destAddr.Address(), time.Second*10, exitIp, 0)
	if err != nil {
		log.Error("[socks_proxy_handler] DialContext 创建目标连接失败", zap.Error(err))
		msg := err.Error()
		resp := socks5.HostUnreachable
		if strings.Contains(msg, "refused") {
			resp = socks5.ConnectionRefused
		} else if strings.Contains(msg, "network is unreachable") {
			resp = socks5.NetworkUnreachable
		}
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			m.dialFailTracker.RecordDialFailConnection(ipAndTarget)
			log.Error("[socks_proxy_handler] tcp dial target timeout!", zap.Any("local_ip", exitIpStr), zap.Any("target_addr", destAddr.Address()), zap.Any("user", user))
		}
		if err = socks5.SendReply(conn, resp, nil); err != nil {
			return
		}
		return
	}
	defer target.Close()

	if !isIpv4 { // 如果不是ipv4，是ipv6
		//告诉客户端连接目标服务器成功
		bind := socks5.AddrSpec{IP: net.ParseIP("0.0.0.0"), Port: 0}
		if err = socks5.SendReply(conn, socks5.SuccessReply, &bind); err != nil {
			return
		}
	}

	clientAddr := conn.RemoteAddr().String()
	log.Info("[socks_proxy_handler] 创建目标连接成功 ",
		zap.Any("username", user),
		zap.Any("s5_proxy_ip", exitIpStr),
		zap.Any("clientAddr", clientAddr),
		zap.Any("destAddr", destAddr.Address()),
	)

	// 告诉客户端连接目标服务器成功
	local := target.LocalAddr().(*net.TCPAddr)
	bind := socks5.AddrSpec{IP: local.IP, Port: local.Port}
	if err = socks5.SendReply(conn, socks5.SuccessReply, &bind); err != nil {
		log.Error("[socks_proxy_handler] 应答socks5.SuccessReply失败", zap.Error(err))
		return
	}

	// key := fmt.Sprintf("%s:%s", user, proxyServerIpStr)
	key := user
	connCtx := m.addUserConnection(key)
	action := connCtx.a
	defer m.deleteUserConnection(key, connCtx)

	netConn := newConn(conn, CONN_WRITE_TIME, CONN_READ_TIME)
	netTarget := newConn(target, CONN_WRITE_TIME, CONN_READ_TIME)

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
			m.ReportAccessLogToInfluxDB(user, *domain, exitIpStr)
		} else {
			hostArr := strings.Split(destAddr.Address(), ":")
			if cap(hostArr) > 0 {
				m.ReportAccessLogToInfluxDB(user, hostArr[0], exitIpStr)
			} else {
				m.ReportAccessLogToInfluxDB(user, destAddr.Address(), exitIpStr)
			}
		}
	}()

	///域名为空，并且使用CONNECT
	if domain == "" {
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
					m.SendBlackListAccessLogMessageData(user, pwd, black, 1, user, exitIpStr)
					log.Error("[socks_proxy_handler] 黑名单定时检测",
						zap.Any("domain", domain),
						zap.Error(err),
						zap.Any("username", user),
						zap.Any("clientAddr", exitIpStr),
						zap.Any("target_host", destAddr.Address()),
					)

					return
				}
			}
		case err, _ := <-errCh:
			if err != nil {
				log.Error("[socks_proxy_handler] conn close!",
					zap.Error(err),
					zap.Any("username", user),
					zap.Any("clientAddr", exitIpStr),
					zap.Any("target_host", destAddr.Address()),
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
