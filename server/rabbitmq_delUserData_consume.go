package server

import (
	"context"
	"fmt"
	"go.uber.org/zap" // 高性能日志库
	"proxy_server/common"
	"proxy_server/config"
	"proxy_server/protobuf"

	"github.com/streadway/amqp"
	"google.golang.org/protobuf/proto"
	"proxy_server/log"
)

// 删除用户消息  // DEL
func (m *manager) runRabbitmqDeleteUserDataQueueConsume(ctx context.Context, conn *amqp.Connection) {
	ch, err := conn.Channel()
	if err != nil {
		log.Error("[rabbitmq_consume] rabbitmq DeleteUserData conn.Channel 错误", zap.Error(err))
		return
	}
	defer ch.Close()

	qName := fmt.Sprintf("DeleteUserData_%s_%s", config.GetConf().LocalIp, config.GetConf().ProcessName)
	// 声明一个队列
	if _, err := ch.QueueDeclare(
		qName, // 队列名称
		true,  // 是否持久化
		false, // 是否自动删除
		false, // 是否排他
		false, // 是否等待服务器响应
		nil,   // 额外参数
	); err != nil {
		log.Error("[rabbitmq_consume] rabbitmq DeleteUserData ch.QueueDeclare 错误", zap.Error(err))
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
		log.Error("[rabbitmq_consume] rabbitmq DeleteUserData ch.Consume 错误", zap.Error(err))
		return
	}

	closeChan := ch.NotifyClose(make(chan *amqp.Error, 1))
	for {
		select {
		case d, ok := <-msgs:
			if ok {
				m.runRabbitmqDeleteUserDataQueueConsumeAction(ctx, d.Body)
			}
		case <-ctx.Done():
			return
		case <-closeChan:
			log.Error("[rabbitmq_consume] rabbitmq DeleteUserData 信道关闭")
			return

		}
	}
}

func (m *manager) runRabbitmqDeleteUserDataQueueConsumeAction(ctx context.Context, body []byte) {
	info := &protobuf.AuthInfo{}
	err := proto.Unmarshal(body, info)
	if err != nil {
		log.Error("[rabbitmq_consume] rabbitmq DeleteUserData proto.Unmarshal 错误", zap.Error(err))
		return
	}

	common.GetRedisDB().Del(ctx, info.Username+":Auth")

	//strKey := fmt.Sprintf("%s_%s", REDIS_AUTH_USERDATA, info.Username)
	//setKey := fmt.Sprintf("%s_%s", REDIS_USER_IPSET, info.Username)
	//
	//// 要删除的键列表
	//keysToDelete := []string{strKey, setKey}
	//
	//// 删除多个键
	//_, err = common.GetRedisDB().Del(context.Background(), keysToDelete...).Result()
	//if err != nil {
	//	log.Error("[rabbitmq_consume] rabbitmq DeleteUserData 删除数据错误", zap.Error(err))
	//	return
	//}
	//
	//for v := range m.userCtxMap.Iter() {
	//	keys := strings.Split(v.Key, ":")
	//	if len(keys) > 0 {
	//		// 判断是否包含username
	//		if keys[0] == info.Username {
	//			v, exist := m.userCtxMap.Pop(v.Key)
	//			if exist && v != nil {
	//				v.cancel()
	//			}
	//		}
	//	}
	//
	//}
}
