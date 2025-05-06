package server

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap" // 高性能日志库

	"proxy_server/config"

	"github.com/streadway/amqp"

	"google.golang.org/protobuf/proto"
	"proxy_server/log"
	"proxy_server/protobuf"
)

// /断开连接
func (m *manager) runRabbitmqDisconnectQueueConsume(ctx context.Context, conn *amqp.Connection) {
	ch, err := conn.Channel()
	if err != nil {
		log.Error("[rabbitmq_consume] rabbitmq Disconnect conn.Channel 错误", zap.Error(err))
		return
	}
	defer ch.Close()

	messageTTL := 60 * 1000 // 一分钟
	args := amqp.Table{
		"x-message-ttl": messageTTL,
	}

	qName := fmt.Sprintf("Disconnect_%s_%s", config.GetConf().LocalIp, config.GetConf().ProcessName)
	// 声明一个队列
	if _, err := ch.QueueDeclare(
		qName, // 队列名称
		true,  // 是否持久化
		false, // 是否自动删除
		false, // 是否排他
		false, // 是否等待服务器响应
		args,  // 额外参数
	); err != nil {
		log.Error("[rabbitmq_consume] rabbitmq Disconnect ch.QueueDeclare 错误", zap.Error(err))
		return
	}

	// 从队列中消费消息
	msgs, err := ch.Consume(
		qName, // 队列名称
		"",    // 消费者名称
		true,  // 是否自动确认
		false, // 是否排他
		false, // 是否为本地队列
		false, // 是否等待服务器响应
		nil,   // 额外参数
	)
	if err != nil {
		log.Error("[rabbitmq_consume] rabbitmq Disconnect ch.Consume 错误", zap.Error(err))
		return
	}

	closeChan := ch.NotifyClose(make(chan *amqp.Error, 1))
	for {
		select {
		case d, ok := <-msgs:
			if ok {
				log.Info("[rabbitmq_consume] rabbitmq Disconnect 接受消息 ")
				m.runRabbitmqDisconnectQueueConsumeAction(d.Body)
			}
		case <-ctx.Done():
			return
		case <-closeChan:
			log.Error("[rabbitmq_consume] rabbitmq Disconnect 信道关闭")
			return

		}
	}
}

func (m *manager) runRabbitmqDisconnectQueueConsumeAction(body []byte) {
	disConnInfo := &protobuf.DisconnectInfo{}
	err := proto.Unmarshal(body, disConnInfo)
	if err != nil {
		log.Error("[rabbitmq_consume] rabbitmq Disconnect proto.Unmarshal 错误", zap.Error(err))
		return
	}

	if disConnInfo.Username == "" {
		log.Error("[rabbitmq_consume] rabbitmq Disconnect disConnInfo.Username 为空", zap.Any("disConnInfo", disConnInfo))
		return
	}

	if len(disConnInfo.Ips) == 0 {
		for v := range m.userCtxMap.Iter() {
			keys := strings.Split(v.Key, ":")
			if len(keys) > 0 {
				// 判断是否包含username
				if keys[0] == disConnInfo.Username {
					v, exist := m.userCtxMap.Pop(v.Key)
					if exist && v != nil {
						v.cancel()
					}
				}
			}

		}
		return
	}

	for _, ip := range disConnInfo.Ips {
		key := fmt.Sprintf("%s:%s", disConnInfo.Username, ip)
		err := m.CloseUserConnections(key)
		if err != nil {
			log.Error("[rabbitmq_consume] rabbitmq Disconnect 根据用户ip关闭连接失败", zap.Error(err))
		}
	}
}
