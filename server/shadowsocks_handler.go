package server

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/shadowsocks/go-shadowsocks2/core"
	"github.com/shadowsocks/go-shadowsocks2/socks"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"proxy_server/log"
	"proxy_server/server/sniffing"
	"proxy_server/server/sniffing/tls"
	util "proxy_server/utils"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const CipherType = "AES-128-GCM"

func (m *manager) runShadowSocks(ctx context.Context, conn net.Conn) {
	// 获取用户入站ip对应的ss的密钥
	ip, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	password, errAuth := m.auth.ValidShadowSocks(ip)
	if errAuth != nil {
		log.Error("[shadowSocks_handler] 鉴权失败", zap.Error(errAuth), zap.Any("ip", ip))
		discardConn(conn)
		return
	}

	// 获取解密Cipherer
	cipher, errPick := core.PickCipher(CipherType, nil, password)
	if errPick != nil {
		log.Error("[shadowSocks_handler] ShadowSocks proxy pick Cipher failed!", zap.Error(errPick))
		discardConn(conn)
		return
	}

	sConn := cipher.StreamConn(conn)

	// 尝试获取目标地址
	tgt, errRead := socks.ReadAddr(sConn)
	if errRead != nil {
		log.Error("[shadowSocks_handler] failed to get target address!", zap.Error(errRead), zap.Any("ip", conn.RemoteAddr()))
		discardConn(conn)
		return
	}

	ipAndTarget := fmt.Sprintf("%s-%s", ip, tgt.String())
	in := m.IpAndTargetIsInBlacklist(ipAndTarget)
	if in {
		msg := fmt.Sprintf("ip[%s]Andserver:[%s] in black list,unable to access!", ip, tgt.String())
		log.Error("[shadowSocks_handler] " + msg)
		return
	}

	if ok, ipCount := m.AddIpConnCount(ip); ok {
		defer m.ReduceIpConnCount(ip)
	} else {
		// ip的连接数到达上限
		log.Error("[shadowSocks_handler] ip连接数达到上限", zap.Any("ip", ip), zap.Any("连接数", ipCount))
		ttl := 300
		alarmOneIPMaxThreshold := m.getNacosConf().AlarmOneIPMaxThreshold
		if alarmOneIPMaxThreshold != 0 {
			ttl = alarmOneIPMaxThreshold
		}
		exist, _ := util.GetOrSetFreeCache(CONNECT_MAX_IP+ip, []byte{1}, ttl)
		if exist == nil {
			util.SendAlarmToDingTalk(fmt.Sprintf("shadowSocks用户，使用ip:%s,达到连接数上限，上限值为%d", ip, alarmOneIPMaxThreshold))
		}
		return
	}

	domain := regexpDomain(tgt.String())
	if domain != "" {
		if black, in := m.IsInBlacklist(domain); in {
			m.SendBlackListAccessLogMessageData(ip, ip, black, 1, "", ip)
			log.Error("[shadowSocks_handler] 黑名单", zap.Any("domain", domain), zap.Any("local_ip", ip), zap.Any("target_addr", black))
			return
		}
	}
	var domainPointer atomic.Pointer[string]
	domainPointer.Store(&domain)

	ipByte := net.ParseIP(ip).To4()
	target, err := DialContext(ctx, "tcp", tgt.String(), time.Second*10, ipByte, 0)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			m.dialFailTracker.RecordDialFailConnection(ipAndTarget)
			log.Error("[shadowSocks_handler] tcp dial target timeout!", zap.Any("local_ip", ip), zap.Any("target_addr", tgt.String()))
		}
		log.Error("[shadowSocks_handler] DialContext 创建目标连接失败", zap.Error(err))
		return
	}
	defer target.Close()

	clientAddr := conn.RemoteAddr().String()
	log.Info("[shadowSocks_handler] 创建目标连接成功 ",
		zap.Any("shadowSocks_handler_ip", ip),
		zap.Any("clientAddr", clientAddr),
		zap.Any("destAddr", tgt.String()),
	)

	user := "shadowSocks"
	key := fmt.Sprintf("%s:%s", user, ip)
	connCtx := m.addUserConnection(key)
	action := connCtx.a
	defer m.deleteUserConnection(key, connCtx)

	netConn := newConn(sConn, CONN_WRITE_TIME, CONN_READ_TIME)
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
			m.ReportAccessLogToInfluxDB(user, *domain, ip)
		} else {
			hostArr := strings.Split(tgt.String(), ":")
			if cap(hostArr) > 0 {
				m.ReportAccessLogToInfluxDB(user, hostArr[0], ip)
			} else {
				m.ReportAccessLogToInfluxDB(user, tgt.String(), ip)
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
					m.SendBlackListAccessLogMessageData(ip, ip, black, 1, "", ip)
					log.Error("[shadowSocks_handler] 黑名单定时检测",
						zap.Any("domain", domain),
						zap.Error(err),
						zap.Any("username", user),
						zap.Any("clientAddr", ip),
						zap.Any("target_host", tgt.String()),
					)

					return
				}
			}
		case err, _ := <-errCh:
			if err != nil {
				log.Error("[shadowSocks_handler] conn close!",
					zap.Error(err),
					zap.Any("username", user),
					zap.Any("clientAddr", ip),
					zap.Any("target_host", tgt.String()),
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

func discardConn(conn net.Conn) {
	_, err := io.Copy(ioutil.Discard, conn)
	if err != nil {
		log.Error("[shadowSocks_handler] discard error", zap.Error(err))
	}
}
