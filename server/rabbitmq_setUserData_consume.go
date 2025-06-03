package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/streadway/amqp"
	"go.uber.org/zap" // 高性能日志库
	"google.golang.org/protobuf/proto"
	"proxy_server/common"
	"proxy_server/config"
	"proxy_server/log"
	"proxy_server/protobuf"
	"proxy_server/server/lua"
)

// /设置用户信息
func (m *manager) runRabbitmqSetUserDataQueueConsume(ctx context.Context, conn *amqp.Connection) {
	ch, err := conn.Channel()
	if err != nil {
		log.Error("[rabbitmq_consume] rabbitmq SetUserData conn.Channel 错误", zap.Error(err))
		return
	}
	defer ch.Close()

	qName := fmt.Sprintf("SetUserData_%s_%s", config.GetConf().LocalIp, config.GetConf().ProcessName)
	// 声明一个队列
	if _, err := ch.QueueDeclare(
		qName, // 队列名称
		true,  // 是否持久化
		false, // 是否自动删除
		false, // 是否排他
		false, // 是否等待服务器响应
		nil,   // 额外参数
	); err != nil {
		log.Error("[rabbitmq_consume] rabbitmq SetUserData ch.QueueDeclare 错误", zap.Error(err))
		return
	}

	// 从队列中消费消息
	msgs, err := ch.Consume(
		qName, // 队列名称
		"",    // 消费者名称
		false, // 是否自动确认
		false, // 是否排他
		false, // 是否为本地队列
		false, // 是否等待服务器响应
		nil,   // 额外参数
	)
	if err != nil {
		log.Error("[rabbitmq_consume] rabbitmq SetUserData ch.Consume 错误", zap.Error(err))
		return
	}

	closeChan := ch.NotifyClose(make(chan *amqp.Error, 1))

	for {
		select {
		case d, ok := <-msgs:
			if ok {
				m.runRabbitmqSetUserDataQueueConsumeAction(ctx, &d)
			}
		case <-ctx.Done():
			return
		case <-closeChan:
			log.Error("[rabbitmq_consume] rabbitmq SetUserData 信道关闭")
			return

		}
	}
}

func (m *manager) runRabbitmqSetUserDataQueueConsumeAction(ctx context.Context, d *amqp.Delivery) {
	info := &protobuf.AuthInfo{}
	err := proto.Unmarshal(d.Body, info)
	if err != nil {
		d.Nack(false, false)
		log.Error("[rabbitmq_consume] rabbitmq SetUserData proto.Unmarshal 错误", zap.Error(err))
		return
	}
	setMembers := []string{}

	for ip := range info.Ips {
		setMembers = append(setMembers, ip)
	}
	info.Ips = nil
	info.UpdateUnix = time.Now().Unix()

	data, err := json.Marshal(info)
	if err != nil {
		d.Nack(false, false)
		log.Error("[rabbitmq_consume] rabbitmq SetUserData json.Marshal 错误", zap.Error(err))
		return
	}

	// 组合 ARGV 参数
	argv := []interface{}{string(data)}
	for _, member := range setMembers {
		argv = append(argv, member)
	}

	// 定义要操作的键和元素
	stringKey := fmt.Sprintf("%s_%s", REDIS_AUTH_USERDATA, info.Username)
	setKey := fmt.Sprintf("%s_%s", REDIS_USER_IPSET, info.Username)

	if _, err := common.GetRedisDB().EvalSha(context.Background(), lua.SetUserDataLuaScriptShaCode(), []string{stringKey, setKey}, argv...).Result(); err != nil {
		log.Error("[rabbitmq_consume] rabbitmq SetUserData 执行lua脚本失败", zap.Error(err))
		d.Nack(false, true)
		return
	}
	d.Ack(false)
	log.Info("[rabbitmq_consume] rabbitmq SetUserData 成功", zap.Any("user", info.Username))
}
