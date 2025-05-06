package server

import (
	"context"
	"fmt"
	"time"

	"github.com/streadway/amqp"
	"go.uber.org/zap" // 高性能日志库
	"proxy_server/config"
	"proxy_server/log"
	"proxy_server/utils/taskConsumerManager"
)

func (m *manager) runRabbitmqConsume(ctx context.Context) {
	amqpUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		config.GetConf().Rabbitmq.Host,
		config.GetConf().Rabbitmq.Port,
		config.GetConf().Rabbitmq.User,
		config.GetConf().Rabbitmq.Password,
		config.GetConf().Rabbitmq.VirtualHost,
	)

	conn, err := amqp.Dial(amqpUrl)
	if err != nil {
		log.Error("[rabbitmq_consume] rabbitmq  amqp.Dial 错误", zap.Error(err))
		timeOutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		<-timeOutCtx.Done()
		return
	}
	defer conn.Close()

	closeChan := conn.NotifyClose(make(chan *amqp.Error, 1))

	tcm := taskConsumerManager.New()
	tcm.AddTask(1, func(ctx context.Context) {
		m.runRabbitmqBlacklistConsume(ctx, conn)
		timeOutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		<-timeOutCtx.Done()
	})

	tcm.AddTask(1, func(ctx context.Context) {
		m.runRabbitmqSetUserDataQueueConsume(ctx, conn)
		timeOutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		<-timeOutCtx.Done()
	})

	tcm.AddTask(1, func(ctx context.Context) {
		m.runRabbitmqDisconnectQueueConsume(ctx, conn)
		timeOutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		<-timeOutCtx.Done()
	})

	tcm.AddTask(1, func(ctx context.Context) {
		m.runRabbitmqDeleteUserDataQueueConsume(ctx, conn)
		timeOutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		<-timeOutCtx.Done()
	})

	select {
	case <-closeChan:
	case <-ctx.Done():
	}

	tcm.Stop()

	timeOutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	<-timeOutCtx.Done()
}
