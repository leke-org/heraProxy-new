package common

import (
	"go.uber.org/zap"
	"proxy_server/log"
	"sync"

	"proxy_server/config"
	"proxy_server/utils/rabbitMQ"
)

// 获取rabbitmq生产者
var GetRabbitMqProductPool = sync.OnceValue[*rabbitMQ.RabbitPool](func() *rabbitMQ.RabbitPool {
	pool := rabbitMQ.NewProductPool()

	err := pool.Connect(config.GetConf().Rabbitmq.Host, config.GetConf().Rabbitmq.Port, config.GetConf().Rabbitmq.User, config.GetConf().Rabbitmq.Password)
	//if err != nil {
	//	panic(err)
	//}
	//err := pool.ConnectVirtualHost(
	//	config.GetConf().Rabbitmq.Host,
	//	config.GetConf().Rabbitmq.Port,
	//	config.GetConf().Rabbitmq.User,
	//	config.GetConf().Rabbitmq.Password,
	//	config.GetConf().Rabbitmq.VirtualHost,
	//)
	if err != nil {
		log.Error("初始化rabbitmq生产者失败:", zap.Error(err))
		//panic(fmt.Errorf("初始化rabbitmq生产者失败error:%+v", err))
	}
	return pool
})

// 获取rabbitmq的消费者
var GetRabbitConsumerPool = sync.OnceValue[*rabbitMQ.RabbitPool](func() *rabbitMQ.RabbitPool {
	pool := rabbitMQ.NewConsumePool()
	err := pool.Connect(config.GetConf().Rabbitmq.Host, config.GetConf().Rabbitmq.Port, config.GetConf().Rabbitmq.User, config.GetConf().Rabbitmq.Password)
	//err := pool.ConnectVirtualHost(
	//	config.GetConf().Rabbitmq.Host,
	//	config.GetConf().Rabbitmq.Port,
	//	config.GetConf().Rabbitmq.User,
	//	config.GetConf().Rabbitmq.Password,
	//	config.GetConf().Rabbitmq.VirtualHost,
	//)
	if err != nil {
		log.Error("初始化rabbitmq消费者失败:", zap.Error(err))
		//panic(fmt.Errorf("初始化rabbitmq消费者失败error:%+v", err))
	}
	return pool
})
