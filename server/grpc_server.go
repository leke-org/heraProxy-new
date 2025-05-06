package server

// import (
// 	"context"
// 	"log"
// 	"net"

// 	"google.golang.org/grpc"

// 	"proxy_server/common"
// 	"proxy_server/config"
// 	"proxy_server/protobuf"
// )

// func (m *manager) initGrpcServer() {
// 	m.grpcServer = grpc.NewServer(
// 		grpc.MaxConcurrentStreams(100*100),
// 		grpc.MaxRecvMsgSize(100*1024*1024),
// 		grpc.MaxSendMsgSize(100*1024*1024),
// 	)
// 	lis, err := net.Listen("tcp", config.GetConf().GrpcListenerAddress)
// 	if err != nil {
// 		log.Panic("[grpc_server] 初始化grpc监听服务失败", zap.Error(err))
// 	}
// 	m.grpcListener = lis
// 	protobuf.RegisterAuthServer(m.grpcServer, m)
// }

// func (m *manager) runGrpcServer(ctx context.Context) {
// 	done := make(chan struct{})
// 	defer func() {
// 		m.grpcServer.GracefulStop()
// 		for range done {
// 			/* code */
// 		}
// 	}()
// 	go func() {
// 		defer close(done)
// 		m.grpcServer.Serve(m.grpcListener)
// 	}()
// 	<-ctx.Done()
// }

// func (s *manager) SetUserData(ctx context.Context, info *protobuf.AuthInfo) (*protobuf.NullMessage, error) {
// 	redisCli := redis.GetRedisClient()
// 	info.UpdateUnix = time.Now().Unix()
// 	data, err := json.Marshal(info)
// 	if err != nil {
// 		log.Error("[grpc_server] SetUserData json错误", zap.Error(err))
// 		return nil, err
// 	}
// 	err = common.GetRedisDB().Set(context.Background(), info.Username+":Auth", string(data), 0).Err()
// 	log.Info("[grpc_server] SetUserData", zap.Any("username", info))

// 	if err != nil {
// 		log.Error("[grpc_server] SetUserData错误", zap.Error(err))
// 		return nil, err
// 	}
// 	return &protobuf.NullMessage{}, nil
// }

// func (s *manager) AddUserData(ctx context.Context, info *protobuf.AuthInfo) (*protobuf.NullMessage, error) {
// 	str, err := common.GetRedisDB().Get(context.Background(), info.Username+":Auth").Result()
// 	authInfo := &protobuf.AuthInfo{}
// 	if err != nil {
// 		authInfo = info
// 	} else {
// 		err := json.Unmarshal([]byte(str), authInfo)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	for k, v := range info.Ips {
// 		authInfo.Ips[k] = v
// 	}
// 	authInfo.UpdateUnix = time.Now().Unix()
// 	data, _ := json.Marshal(authInfo)
// 	err = common.GetRedisDB().Set(ctx, authInfo.Username+":Auth", string(data), 0).Err()
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &protobuf.NullMessage{}, nil
// }

// func (s *manager) DeleteUserData(ctx context.Context, info *protobuf.AuthInfo) (*protobuf.NullMessage, error) {
// 	common.GetRedisDB().Del(ctx, info.Username+":Auth")
// 	logrus.WithFields(logrus.Fields{"username": info.Username}).WithFields(logrus.Fields{"AuthServer": "delete"}).Info("删除操作")
// 	return &protobuf.NullMessage{}, nil
// }

// func (s *manager) GetUserData(ctx context.Context, info *protobuf.AuthInfo) (*protobuf.AuthInfo, error) {
// 	storeKey := info.Username + ":Auth"
// 	str, err := common.GetRedisDB().Get(ctx, storeKey).Result()
// 	if err != nil {
// 		return nil, err
// 	}
// 	authInfo := &protobuf.AuthInfo{}
// 	err = json.Unmarshal([]byte(str), authInfo)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return authInfo, nil
// }

// func (s *manager) Disconnect(ctx context.Context, info *protobuf.DisconnectInfo) (*protobuf.NullMessage, error) {
// 	logrus.WithFields(logrus.Fields{"username": info.Username}).Info("close user all connect ")
// 	redisCli := common.GetRedisDB().GetRedisClient()
// 	channel := "disconnect"
// 	data, err := json.Marshal(info)
// 	if err != nil {
// 		logrus.WithFields(logrus.Fields{"username": info.Username}).Error("call AuthServer.Disconnect failed! ")
// 		return nil, err
// 	}
// 	err = common.GetRedisDB().Publish(ctx, channel, string(data)).Err()
// 	if err != nil {
// 		logrus.WithFields(logrus.Fields{"username": info.Username}).Error("publish message failed! ")
// 		return nil, err
// 	}
// 	return &protobuf.NullMessage{}, nil
// }
