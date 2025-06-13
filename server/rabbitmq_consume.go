package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap" // 高性能日志库
	"google.golang.org/protobuf/proto"
	"proxy_server/common"
	"proxy_server/config"
	"proxy_server/log"
	"proxy_server/protobuf"
	"proxy_server/utils/rabbitMQ"
)

const (
	AuthEventIpv4SetUserData         = 1
	AuthEventIpv4DeleteUserData      = 2
	AuthEventIpv4AddUserData         = 3
	AuthEventIpv4DisconnectUserData  = 4
	AuthEventIpv6SetUserData         = 5
	AuthEventIpv6SetUserDataBatch    = 6
	AuthEventIpv6DeleteUserData      = 7
	AuthEventIpv6DeleteUserDataBatch = 8
	AuthEventIpv6BanUserData         = 9
	AuthEventIpv6BanUserDataBatch    = 10
	AuthEventAddShadowSocksData      = 11
	AuthEventDeleteShadowSocksData   = 12
	AuthEventIpv6SEOData             = 13
)

func (m *manager) runRabbitmqConsume(ctx context.Context) {
	consumeBlacklist := &rabbitMQ.ConsumeReceive{ // 黑名单消费者
		ExchangeName: config.GetConf().Rabbitmq.BlacklistExchange, // 交换机名称
		ExchangeType: rabbitMQ.EXCHANGE_TYPE_FANOUT,
		Route:        "",
		QueueName:    "blackListMsg_" + config.GetConf().LocalIp + "_" + config.GetConf().ProcessName,
		IsTry:        false, // 是否重试
		IsAutoAck:    true,  // 自动消息确认
		MaxReTry:     3,     // 最大重试次数
		EventFail: func(code int, e error, data []byte) {
			log.Error("[rabbitmq_consume] rabbitmq  consumeBlacklistMsg 错误", zap.Error(e), zap.Any("code:", code), zap.Any("data", string(data)))
		},
		EventSuccess: func(data []byte, header map[string]interface{}, retryClient rabbitMQ.RetryClientInterface) bool { // 如果返回true 则无需重试
			m.runRabbitmqBlacklistConsumeAction(data)
			return true
		},
	}

	consumeAuthEvent := &rabbitMQ.ConsumeReceive{ // 黑名单消费者
		// ExchangeName: config.GetConf().Rabbitmq.AuthInfoExchange, // 交换机名称
		ExchangeType: rabbitMQ.EXCHANGE_TYPE_DIRECT,
		Route:        "",
		QueueName:    "authEventMsg_" + config.GetConf().LocalIp + "_" + config.GetConf().ProcessName,
		IsTry:        true, // 是否重试
		IsAutoAck:    true, // 自动消息确认
		MaxReTry:     3,    // 最大重试次数
		EventFail: func(code int, e error, data []byte) {
			log.Error("[rabbitmq_consume] rabbitmq  consumeAuthEventMsg 错误", zap.Error(e), zap.Any("code:", code), zap.Any("data", string(data)))
		},
		EventSuccess: func(data []byte, header map[string]interface{}, retryClient rabbitMQ.RetryClientInterface) bool { // 如果返回true 则无需重试
			err := m.runRabbitmqAuthEventConsumeAction(ctx, data)
			if err != nil {
				return false
			}
			return true
		},
	}

	common.GetRabbitConsumerPool().RegisterConsumeReceive(consumeBlacklist)
	common.GetRabbitConsumerPool().RegisterConsumeReceive(consumeAuthEvent)

	err := common.GetRabbitConsumerPool().RunConsume()
	if err != nil {
		log.Error("[rabbitmq_consume] start consumeCloseUserConn fail", zap.Error(err))
		panic(err)
	}
}

func (m *manager) runRabbitmqAuthEventConsumeAction(ctx context.Context, body []byte) error {
	authEvent := protobuf.AuthEvent{}
	err := proto.Unmarshal(body, &authEvent)
	if err != nil {
		log.Error("[rabbitmq_authEvent_consume] rabbitmq  authEvent proto.Unmarshal 错误", zap.Error(err))
		return err
	}
	switch authEvent.Type {
	case AuthEventIpv4SetUserData:
		err = m.handlerAuthEventIpv4SetUserData(ctx, authEvent.GetAuthInfo())
	case AuthEventIpv4DeleteUserData:
		err = m.handlerAuthEventIpv4DeleteUserData(ctx, authEvent.GetAuthInfo())
	case AuthEventIpv4AddUserData:
		err = m.handlerAuthEventIpv4AddUserData(ctx, authEvent.GetAuthInfo())
	case AuthEventIpv4DisconnectUserData:
		err = m.handlerAuthEventIpv4DisconnectUserData(ctx, authEvent.GetDisconnectInfo())
	case AuthEventIpv6SetUserData:
		err = m.handlerAuthEventIpv6SetUserData(ctx, authEvent.GetIpv6AuthInfo())
	case AuthEventIpv6SetUserDataBatch:
		err = m.handlerAuthEventIpv6SetUserDataBatch(ctx, authEvent.GetIpv6AuthInfoList())
	case AuthEventIpv6DeleteUserData:
		err = m.handlerAuthEventIpv6DeleteUserData(ctx, authEvent.GetIpv6AuthInfo())
	case AuthEventIpv6DeleteUserDataBatch:
		err = m.handlerAuthEventIpv6DeleteUserDataBatch(ctx, authEvent.GetIpv6AuthInfoList())
	case AuthEventIpv6BanUserData:
		err = m.handlerAuthEventIpv6BanUserData(ctx, authEvent.GetIpv6AuthInfo())
	case AuthEventIpv6BanUserDataBatch:
		err = m.handlerAuthEventIpv6BanUserDataBatch(ctx, authEvent.GetIpv6AuthInfoList())
	case AuthEventAddShadowSocksData:
		err = m.handlerAuthEventAddShadowSocksData(ctx, authEvent.GetShadowSocksData())
	case AuthEventDeleteShadowSocksData:
		err = m.handlerAuthEventDeleteShadowSocksData(ctx, authEvent.GetShadowSocksData())
	case AuthEventIpv6SEOData:
		err = m.handlerAuthEventIpv6SEOData(ctx, authEvent.GetIpv6GeoInfo())
	}
	return err
}

func (m *manager) handlerAuthEventIpv4SetUserData(ctx context.Context, authInfo *protobuf.AuthInfo) error {
	if authInfo == nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv4SetUserData authInfo is nil !")
		return errors.New("authInfo is nil")
	}
	if authInfo.Username == "" {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv4SetUserData authInfo.username is nil !")
		return errors.New("authInfo.username is nil")
	}
	data, err := json.Marshal(authInfo)
	if err != nil {
		log.Error("[rabbitmq_authEvent_consume] proto.Marshal err !", zap.Error(err))
		return err
	}
	err = common.GetRedisDB().Set(ctx, authInfo.Username+":Auth", data, 0).Err()
	if err != nil {
		log.Error("[rabbitmq_consume] Redis set auth key err", zap.Error(err))
		return err
	}
	log.Info("[rabbitmq_consume] rabbitmq SetUserData 成功", zap.Any("user", authInfo.Username))
	return nil
}

func (m *manager) handlerAuthEventIpv4DeleteUserData(ctx context.Context, authInfo *protobuf.AuthInfo) error {
	if authInfo == nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv4DeleteUserData authInfo is nil !")
		return errors.New("authInfo is nil")
	}
	if authInfo.Username == "" {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv4DeleteUserData authInfo.username is nil !")
		return errors.New("authInfo.username is nil")
	}
	err := common.GetRedisDB().Del(ctx, authInfo.Username+":Auth").Err()
	if err != nil {
		log.Error("[rabbitmq_consume] Redis del Ipv4 auth key err", zap.Error(err))
	}
	log.Info("[rabbitmq_consume] rabbitmq DeleteUserData 成功", zap.Any("user", authInfo.Username))
	return nil
}

func (m *manager) handlerAuthEventIpv4AddUserData(ctx context.Context, authInfo *protobuf.AuthInfo) error {
	if authInfo == nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv4DeleteUserData authInfo is nil !")
		return errors.New("authInfo is nil")
	}
	if authInfo.Username == "" {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv4DeleteUserData authInfo.username is nil !")
		return errors.New("authInfo.username is nil")
	}
	str, errGet := common.GetRedisDB().Get(ctx, authInfo.Username+":Auth").Result()
	saveInfo := &protobuf.AuthInfo{}
	if errGet != nil {
		saveInfo = authInfo
	} else {
		err := json.Unmarshal([]byte(str), saveInfo)
		if err != nil {
			log.Error("[rabbitmq_consume] rabbitmq AddUserData proto.Unmarshal 错误", zap.Error(err))
			return err
		}
	}

	for k, v := range authInfo.Ips {
		saveInfo.Ips[k] = v
	}
	saveInfo.UpdateUnix = time.Now().Unix()
	data, _ := json.Marshal(saveInfo)
	err := common.GetRedisDB().Set(ctx, saveInfo.Username+":Auth", string(data), 0).Err()
	if err != nil {
		log.Error("[rabbitmq_consume] Redis set auth key err", zap.Error(err))
		return err
	}
	log.Info("[rabbitmq_consume] rabbitmq Ipv4AddUserData 成功", zap.Any("user", saveInfo.Username))
	return nil
}

func (m *manager) handlerAuthEventIpv4DisconnectUserData(ctx context.Context, disconnectInfo *protobuf.DisconnectInfo) error {
	if disconnectInfo == nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv4DisconnectUserData disconnectInfo is nil !")
		return errors.New("disconnectInfo is nil")
	}
	if disconnectInfo.Username == "" {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv4DisconnectUserData disconnectInfo.username is nil !")
		return errors.New("disconnectInfo.username is nil")
	}

	if len(disconnectInfo.Ips) == 0 {
		for v := range m.userCtxMap.Iter() {
			keys := strings.Split(v.Key, ":")
			if len(keys) > 0 {
				// 判断是否包含username
				if keys[0] == disconnectInfo.Username {
					v, exist := m.userCtxMap.Pop(v.Key)
					if exist && v != nil {
						v.cancel()
					}
				}
			}

		}
		return nil
	}

	for _, ip := range disconnectInfo.Ips {
		key := fmt.Sprintf("%s:%s", disconnectInfo.Username, ip)
		err := m.CloseUserConnections(key)
		if err != nil {
			log.Error("[rabbitmq_consume] rabbitmq Disconnect 根据用户ip关闭连接失败", zap.Error(err))
		}
	}
	return nil
}

func (m *manager) handlerAuthEventIpv6SetUserData(ctx context.Context, ipv6AuthInfo *protobuf.Ipv6AuthInfo) error {
	if ipv6AuthInfo == nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6SetUserData ipv6AuthInfo is nil !")
		return errors.New("ipv6AuthInfo is nil")
	}
	if ipv6AuthInfo.Username == "" {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6SetUserData ipv6AuthInfo.username is nil !")
		return errors.New("ipv6AuthInfo.username is nil")
	}

	data, err := proto.Marshal(ipv6AuthInfo)
	if err != nil {
		log.Error("[rabbitmq_authEvent_consume] proto.Marshal err !", zap.Error(err))
		return err
	}
	err = common.GetRedisDB().Set(ctx, ipv6AuthInfo.Username+":Auth", data, 0).Err()
	if err != nil {
		log.Error("[rabbitmq_consume] Redis set Ipv6Auth key err", zap.Error(err))
		return err
	}
	log.Info("[rabbitmq_consume] rabbitmq SetUserIpv6Data 成功", zap.Any("user", ipv6AuthInfo.Username))
	return nil
}

func (m *manager) handlerAuthEventIpv6SetUserDataBatch(ctx context.Context, ipv6AuthInfoList *protobuf.Ipv6AuthInfoList) error {
	if ipv6AuthInfoList == nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6SetUserDataBatch ipv6AuthInfoList is nil !")
		return errors.New("ipv6AuthInfo is nil")
	}
	if len(ipv6AuthInfoList.Ipv6AuthInfoList) <= 0 {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6SetUserDataBatch ipv6AuthInfoList.Ipv6AuthInfoList is empty !")
		return errors.New("ipv6AuthInfoList.Ipv6AuthInfoList is empty")
	}

	redisCli := common.GetRedisDB()
	pipe := redisCli.Pipeline()
	for i := 0; i < len(ipv6AuthInfoList.Ipv6AuthInfoList); i++ {
		data, err := proto.Marshal(ipv6AuthInfoList.Ipv6AuthInfoList[i])
		if err != nil {
			return err
		}
		pipe.Set(context.Background(), ipv6AuthInfoList.Ipv6AuthInfoList[i].Username+":Auth", string(data), 0)
	}
	cmder, err := pipe.Exec(ctx)
	if err != nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6SetUserDataBatch pipe.Exec err !", zap.Error(err))
		return err
	}
	for i := 0; i < len(ipv6AuthInfoList.Ipv6AuthInfoList); i++ {
		if cmder[i].Err() != nil {
			log.Error("[rabbitmq_authEvent_consume] one of handlerAuthEventIpv6SetUserDataBatch  err !", zap.Error(cmder[i].Err()))
		}
	}
	log.Info("[rabbitmq_consume] rabbitmq SetUserIpv6DataBatch 成功", zap.Any("data", ipv6AuthInfoList.Ipv6AuthInfoList))
	return nil
}

func (m *manager) handlerAuthEventIpv6DeleteUserData(ctx context.Context, ipv6AuthInfo *protobuf.Ipv6AuthInfo) error {
	if ipv6AuthInfo == nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6DeleteUserData ipv6AuthInfo is nil !")
		return errors.New("ipv6AuthInfo is nil")
	}
	if ipv6AuthInfo.Username == "" {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6DeleteUserData ipv6AuthInfo.username is nil !")
		return errors.New("ipv6AuthInfo.username is nil")
	}

	err := common.GetRedisDB().Del(ctx, ipv6AuthInfo.Username+":Auth").Err()
	if err != nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6DeleteUserData  err !", zap.Error(err))
		return err
	}

	log.Info("[rabbitmq_consume] rabbitmq Ipv6DeleteUserData 成功", zap.Any("user", ipv6AuthInfo.Username))
	return nil
}

func (m *manager) handlerAuthEventIpv6DeleteUserDataBatch(ctx context.Context, ipv6AuthInfoList *protobuf.Ipv6AuthInfoList) error {
	if ipv6AuthInfoList == nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6DeleteUserDataBatch ipv6AuthInfoList is nil !")
		return errors.New("ipv6AuthInfo is nil")
	}
	if len(ipv6AuthInfoList.Ipv6AuthInfoList) <= 0 {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6DeleteUserDataBatch ipv6AuthInfoList.Ipv6AuthInfoList is empty !")
		return errors.New("ipv6AuthInfoList.Ipv6AuthInfoList is empty")
	}

	pipe := common.GetRedisDB().Pipeline()
	for i := 0; i < len(ipv6AuthInfoList.Ipv6AuthInfoList); i++ {
		pipe.Del(ctx, ipv6AuthInfoList.Ipv6AuthInfoList[i].Username+":Auth")
	}
	cmder, err := pipe.Exec(ctx)
	if err != nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6DeleteUserDataBatch pipe.Exec err !", zap.Error(err))
		return err
	}
	for i := 0; i < len(ipv6AuthInfoList.Ipv6AuthInfoList); i++ {
		if cmder[i].Err() != nil {
			log.Error("[rabbitmq_authEvent_consume] one of handlerAuthEventIpv6DeleteUserDataBatch  err !", zap.Error(cmder[i].Err()))
		}
	}
	log.Info("[rabbitmq_consume] rabbitmq DeleteUserIpv6DataBatch 成功", zap.Any("data", ipv6AuthInfoList.Ipv6AuthInfoList))
	return nil
}

func (m *manager) handlerAuthEventIpv6BanUserData(ctx context.Context, ipv6AuthInfo *protobuf.Ipv6AuthInfo) error {
	if ipv6AuthInfo == nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6BanUserData ipv6AuthInfo is nil !")
		return errors.New("ipv6AuthInfo is nil")
	}
	if ipv6AuthInfo.Username == "" {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6BanUserData ipv6AuthInfo.username is nil !")
		return errors.New("ipv6AuthInfo.username is nil")
	}

	ipv6AuthInfo.Banned = true
	data, _ := proto.Marshal(ipv6AuthInfo)
	err := common.GetRedisDB().Set(ctx, ipv6AuthInfo.Username+":Auth", string(data), 0).Err()
	m.CloseUserConnections(ipv6AuthInfo.Username)

	if err != nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6BanUserData  set  ipv6AuthInfo err !", zap.Error(err))
		return err
	}
	log.Info("[rabbitmq_consume] rabbitmq handlerAuthEventIpv6BanUserData 成功", zap.Any("user", ipv6AuthInfo.Username))
	return nil
}

func (m *manager) handlerAuthEventIpv6BanUserDataBatch(ctx context.Context, ipv6AuthInfoList *protobuf.Ipv6AuthInfoList) error {
	if ipv6AuthInfoList == nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6DeleteUserDataBatch ipv6AuthInfoList is nil !")
		return errors.New("ipv6AuthInfo is nil")
	}
	if len(ipv6AuthInfoList.Ipv6AuthInfoList) <= 0 {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6DeleteUserDataBatch ipv6AuthInfoList.Ipv6AuthInfoList is empty !")
		return errors.New("ipv6AuthInfoList.Ipv6AuthInfoList is empty")
	}

	pipe := common.GetRedisDB().Pipeline()
	for i := 0; i < len(ipv6AuthInfoList.Ipv6AuthInfoList); i++ {
		ipv6AuthInfoList.Ipv6AuthInfoList[i].Banned = true
		data, err := proto.Marshal(ipv6AuthInfoList.Ipv6AuthInfoList[i])
		if err != nil {
			log.Error("[rabbitmq_authEvent_consume] proto.Marshal err !", zap.Error(err))
			return err
		}
		pipe.Set(ctx, ipv6AuthInfoList.Ipv6AuthInfoList[i].Username+":Auth", string(data), 0)
	}
	cmder, err := pipe.Exec(ctx)
	if err != nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6BanUserDataBatch pipe.Exec err !", zap.Error(err))
		return err
	}
	for i := 0; i < len(ipv6AuthInfoList.Ipv6AuthInfoList); i++ {
		if cmder[i].Err() != nil {
			log.Error("[rabbitmq_authEvent_consume] one of handlerAuthEventIpv6BanUserDataBatch  err !", zap.Error(cmder[i].Err()))
		}
	}
	for i := 0; i < len(ipv6AuthInfoList.Ipv6AuthInfoList); i++ {
		m.CloseUserConnections(ipv6AuthInfoList.Ipv6AuthInfoList[i].Username)
	}

	log.Info("[rabbitmq_consume] rabbitmq Ipv6DeleteUserData 成功", zap.Any("user", ipv6AuthInfoList.Ipv6AuthInfoList))
	return nil
}

func (m *manager) handlerAuthEventAddShadowSocksData(ctx context.Context, data *protobuf.ShadowSocksData) error {
	if data == nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventAddShadowSocksData data is nil !")
		return errors.New("data is nil")
	}
	if data.Ip == "" {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventAddShadowSocksData data.ip is nil !")
		return errors.New("data.ip is nil")
	}

	err := common.GetRedisDB().Set(ctx, data.Ip+":ShadowSocks", data.Password, 0).Err()
	if err != nil {
		log.Error("[rabbitmq_consume] Redis set auth key err", zap.Error(err))
		return err
	}
	log.Info("[rabbitmq_consume] rabbitmq AddShadowSocksData 成功", zap.Any("ip", data.Ip))
	return nil
}

func (m *manager) handlerAuthEventDeleteShadowSocksData(ctx context.Context, data *protobuf.ShadowSocksData) error {
	if data == nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventDeleteShadowSocksData data is nil !")
		return errors.New("data is nil")
	}
	if data.Ip == "" {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventDeleteShadowSocksData data.ip is nil !")
		return errors.New("data.ip is nil")
	}

	err := common.GetRedisDB().Del(ctx, data.Ip+":ShadowSocks").Err()
	if err != nil {
		log.Error("[rabbitmq_consume] Redis Del auth key err", zap.Error(err))
		return err
	}
	log.Info("[rabbitmq_consume] rabbitmq DeleteShadowSocksData 成功", zap.Any("ip", data.Ip))
	return nil
}

func (m *manager) handlerAuthEventIpv6SEOData(ctx context.Context, data *protobuf.Ipv6GEOInfo) error {
	if data == nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6SEOData Ipv6GEOInfo is nil !")
		return errors.New("authInfo is nil")
	}
	if len(data.CitySeg) <= 0 {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6SEOData Ipv6GEOInfo.CitySeg is empty !")
		return errors.New("Ipv6GEOInfo.CitySeg is empty")
	}

	pipe := common.GetRedisDB().Pipeline()
	for k, v := range data.CitySeg {
		pipe.Set(ctx, CitySegPrefix+k, v, 0)
		pipe.SAdd(ctx, AllCitySegSetPrefix, v)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Error("[rabbitmq_authEvent_consume] handlerAuthEventIpv6SEOData pipe.Exec err !", zap.Error(err))
		return err
	}

	log.Info("[rabbitmq_consume] handlerAuthEventIpv6SEOData 成功")
	return nil
}

//func (m *manager) runRabbitmqConsumeIpv6(ctx context.Context) {
//	amqpUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
//		config.GetConf().Rabbitmq.Host,
//		config.GetConf().Rabbitmq.Port,
//		config.GetConf().Rabbitmq.User,
//		config.GetConf().Rabbitmq.Password,
//		config.GetConf().Rabbitmq.VirtualHost,
//	)
//
//	conn, err := amqp.Dial(amqpUrl)
//	if err != nil {
//		log.Error("[rabbitmq_consume] rabbitmq  amqp.Dial 错误", zap.Error(err))
//		timeOutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
//		defer cancel()
//		<-timeOutCtx.Done()
//		return
//	}
//	defer conn.Close()
//
//	closeChan := conn.NotifyClose(make(chan *amqp.Error, 1))
//
//	tcm := taskConsumerManager.New()
//	tcm.AddTask(1, func(ctx context.Context) {
//		m.runRabbitmqBlacklistConsume(ctx, conn)
//		timeOutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
//		defer cancel()
//		<-timeOutCtx.Done()
//	})
//
//	tcm.AddTask(1, func(ctx context.Context) {
//		m.runRabbitmqSetUserIpv6DataQueueConsume(ctx, conn)
//		timeOutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
//		defer cancel()
//		<-timeOutCtx.Done()
//	})
//
//	tcm.AddTask(1, func(ctx context.Context) {
//		m.runRabbitmqDisconnectQueueConsume(ctx, conn)
//		timeOutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
//		defer cancel()
//		<-timeOutCtx.Done()
//	})
//
//	tcm.AddTask(1, func(ctx context.Context) {
//		m.runRabbitmqDeleteUserIpv6DataQueueConsume(ctx, conn)
//		timeOutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
//		defer cancel()
//		<-timeOutCtx.Done()
//	})
//
//	select {
//	case <-closeChan:
//	case <-ctx.Done():
//	}
//
//	tcm.Stop()
//
//	timeOutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
//	defer cancel()
//	<-timeOutCtx.Done()
//}
