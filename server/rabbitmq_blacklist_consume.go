package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/streadway/amqp"
	"go.uber.org/zap" // 高性能日志库
	"proxy_server/config"
	"proxy_server/log"
)

func (m *manager) runRabbitmqBlacklistConsume(ctx context.Context, conn *amqp.Connection) {
	ch, err := conn.Channel()
	if err != nil {
		log.Error("[rabbitmq_blacklist_consume] rabbitmq 黑名单 conn.Channel 错误", zap.Error(err))
		return
	}
	defer ch.Close()
	qName := fmt.Sprintf("new_blackListMsg_%s_%s", config.GetConf().LocalIp, config.GetConf().ProcessName)
	// 声明一个队列
	if _, err := ch.QueueDeclare(
		qName, // 队列名称
		true,  // 是否持久化
		false, // 是否自动删除
		false, // 是否排他
		false, // 是否等待服务器响应
		nil,   // 额外参数
	); err != nil {
		log.Error("[rabbitmq_blacklist_consume] rabbitmq 黑名单 ch.QueueDeclare 错误", zap.Error(err))
		return
	}

	if err := ch.QueueBind(
		qName, // 队列名称
		"",    // FANOUT 类型交换机忽略路由键，这里为空字符串
		config.GetConf().Rabbitmq.BlacklistExchange, // 交换机名称
		false, // 是否等待服务器响应
		nil,   // 额外参数
	); err != nil {
		log.Error("[rabbitmq_blacklist_consume] rabbitmq 黑名单 ch.QueueBind 错误", zap.Error(err))
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
		log.Error("[rabbitmq_blacklist_consume] rabbitmq 黑名单 ch.Consume 错误", zap.Error(err))
		return
	}

	closeChan := ch.NotifyClose(make(chan *amqp.Error, 1))

	for {
		select {
		case d, ok := <-msgs:
			if ok {
				m.runRabbitmqBlacklistConsumeAction(d.Body)
			}
		case <-ctx.Done():
			return
		case <-closeChan:
			log.Error("[rabbitmq_blacklist_consume] rabbitmq 黑名单 信道关闭")
			return

		}
	}
}

func (m *manager) runRabbitmqBlacklistConsumeAction(body []byte) {
	blacklistMsg := BlacklistBroadcastMsg{}
	err := json.Unmarshal(body, &blacklistMsg)
	if err != nil {
		log.Error("[rabbitmq_blacklist_consume] rabbitmq 黑名单 json.Unmarshal 错误", zap.Error(err))
		return
	}

	m.SetBlackMap(blacklistMsg.Blacklist)
}
