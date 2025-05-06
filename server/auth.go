package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"proxy_server/common"
	"proxy_server/protobuf"
)

func (m *manager) Valid(ctx context.Context, username, password, ip string) (authInfo *protobuf.AuthInfo, resErr error) {
	// 创建管道
	pipe := common.GetRedisDB().Pipeline()

	// 定义要操作的键和元素
	strKey := fmt.Sprintf("%s_%s", REDIS_AUTH_USERDATA, username)
	setKey := fmt.Sprintf("%s_%s", REDIS_USER_IPSET, username)

	// 添加获取字符串值和判断集合元素是否存在的操作到管道
	getOp := pipe.Get(ctx, strKey)
	sismemberOp := pipe.SIsMember(ctx, setKey, ip)

	// 执行管道操作
	_, err := pipe.Exec(ctx)
	if err != nil {
		resErr = fmt.Errorf("redis管道命令执行失败 error:%+v", err)
		return
	}

	// 处理获取字符串值的结果
	strVal, err := getOp.Result()
	if err != nil {
		if err == redis.Nil {
			resErr = fmt.Errorf("%s用户数据不存在", username)
			return
		} else {
			resErr = fmt.Errorf("获取%s用户数据失败 error:%+v", username, err)
			return
		}
	}

	authInfo = &protobuf.AuthInfo{}
	err = json.Unmarshal([]byte(strVal), authInfo)
	if err != nil {
		resErr = fmt.Errorf("json.Unmarshal解析%s用户数据%s失败 error:%+v", username, strVal, err)
		return
	}

	if authInfo.Username != username || authInfo.Password != password {
		resErr = fmt.Errorf("%s用户密码错误 用户数据%+v", username, authInfo)
		return
	}

	// 处理判断集合元素是否存在的结果
	exists, err := sismemberOp.Result()
	if err != nil {
		resErr = fmt.Errorf("检测%s用户ip:%+v 执行命令失败 error:%+v", username, ip, err)
		return
	}

	if !exists {
		resErr = fmt.Errorf("检测%s用户ip:%+v不存在", username, ip)
		return
	}

	return authInfo, nil
}
