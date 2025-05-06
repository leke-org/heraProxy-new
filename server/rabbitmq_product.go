package server

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap" // 高性能日志库
	"google.golang.org/protobuf/proto"
	"proxy_server/common"
	"proxy_server/config"
	"proxy_server/log"
	"proxy_server/protobuf"
	"proxy_server/utils"
	"proxy_server/utils/Queue"
	"proxy_server/utils/rabbitMQ"
)

const (
	RabbitmqSendQueueSlicesShareSize    = 8
	RabbitmqSendQueueSlicesHandlerCount = 4 //
)

func (m *manager) initRabbitmqSendQueueSlices() {
	for i := 0; i < RabbitmqSendQueueSlicesShareSize; i++ {
		m.rabbitmqSendQueueSlices = append(m.rabbitmqSendQueueSlices, Queue.NewMQueue[*rabbitMQ.RabbitMqData]())
	}
	m.rabbitmqSendQueueDone = make(chan struct{})
	go func() {
		defer close(m.rabbitmqSendQueueDone)
		m.runhandleRabbitmqSendQueue()
	}()
}

func (m *manager) closeRabbitmqSendQueueSlices() {
	for i := 0; i < RabbitmqSendQueueSlicesShareSize; i++ {
		m.rabbitmqSendQueueSlices[i].Close()
	}
	<-m.rabbitmqSendQueueDone
}

func (m *manager) pushRabbitmqSendQueue(data *rabbitMQ.RabbitMqData) {
	m.rabbitmqSendQueueSlices[m.rabbitmqSendQueueSlicesCounter.Add(1)%RabbitmqSendQueueSlicesShareSize].Enqueue(data)
}

func (m *manager) runhandleRabbitmqSendQueue() {
	wg := sync.WaitGroup{}
	defer wg.Wait()
	for i := 0; i < RabbitmqSendQueueSlicesShareSize; i++ {
		wg.Add(1)
		go func(share int) {
			defer wg.Done()
			m.handleRabbitmqSendQueue(share)
		}(i)
	}
}

func (m *manager) handleRabbitmqSendQueue(share int) {
	queue := m.rabbitmqSendQueueSlices[share]
	wg := sync.WaitGroup{}
	defer wg.Wait()
	for i := 0; i < RabbitmqSendQueueSlicesHandlerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 定义出队处理函数
			processFunc := func(pkg *rabbitMQ.RabbitMqData, isClose bool) bool {
				err := common.GetRabbitMqProductPool().Push(pkg)
				if err != nil {
					log.Error("[rabbitmq_product] send data to rabbitmq 错误", zap.Error(err))
				}
				return true
			}

			// 从队列中取出并处理数据包
			queue.DequeueFunc(processFunc)
		}()
	}
}

func (m *manager) SendAccessLogMessageToInfluxDB(body []byte) {
	unique := time.Now().String() + util.RandStringBytesMaskImprSrcSB(8)
	data := rabbitMQ.GetRabbitMqDataFormat("", "", config.GetConf().Rabbitmq.AccesslogToInfluxDBQueue, "", body, unique)
	m.pushRabbitmqSendQueue(data)
}

func (m *manager) SendBlackListAccessLogMessageData(proxyUserName, proxyServerIpStr, black string, AccountType int32, Account string, ExitIp string) error {
	blackListAccessLog := &protobuf.BlackListAccessLog{
		Site:        black,
		AccountType: AccountType,
		Account:     proxyUserName,
		ExitIp:      proxyServerIpStr,
	}

	sendByte, err := proto.Marshal(blackListAccessLog)
	if err == nil {
		m.SendBlackListAccessLogMessage(sendByte)
	} else {
		return fmt.Errorf("序列化黑名单访问记录失败%w", err)
	}
	return nil
}

func (m *manager) SendBlackListAccessLogMessage(body []byte) {
	unique := time.Now().String() + util.RandStringBytesMaskImprSrcSB(8)
	data := rabbitMQ.GetRabbitMqDataFormat("", "", config.GetConf().Rabbitmq.BlacklistAccesslogQueue, "", body, unique)
	m.pushRabbitmqSendQueue(data)
}

func (m *manager) ReportAccessLogToInfluxDB(user, domain, ip string) {
	accessLog := protobuf.AccessRecordsToInfluxDB{
		UserName:  user,
		Domain:    domain,
		Ip:        ip,
		ProxyType: "ipv4",
	}
	sendBytes, _ := proto.Marshal(&accessLog)
	m.SendAccessLogMessageToInfluxDB(sendBytes)
}
