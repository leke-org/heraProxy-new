package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jellydator/ttlcache/v3"
	"google.golang.org/protobuf/proto"
	"time"

	"proxy_server/common"
	"proxy_server/protobuf"
)

const AuthTtl = 30

type Auth interface {
	Valid(ctx context.Context, username, password, ip string) (exitIp string, resErr error)
	ValidShadowSocks(ipAddr string) (password string, resErr error)
}

type Ipv4Auth struct {
	cache            *ttlcache.Cache[string, *protobuf.AuthInfo]
	shadowSocksCache *ttlcache.Cache[string, string]
}

type Ipv6Auth struct {
	cache *ttlcache.Cache[string, *protobuf.Ipv6AuthInfo]
}

func NewIpv4Auth() *Ipv4Auth {
	r := &Ipv4Auth{}
	r.cache = ttlcache.New[string, *protobuf.AuthInfo](
		ttlcache.WithTTL[string, *protobuf.AuthInfo](AuthTtl*time.Second),
		ttlcache.WithDisableTouchOnHit[string, *protobuf.AuthInfo](), // 不自动延期
	)
	r.shadowSocksCache = ttlcache.New[string, string](
		ttlcache.WithTTL[string, string](AuthTtl*time.Second),
		ttlcache.WithDisableTouchOnHit[string, string](), // 不自动延期
	)
	go r.cache.Start() //开始自动删除过期项目
	go r.shadowSocksCache.Start()
	return r
}

func NewIpv6Auth() *Ipv6Auth {
	r := &Ipv6Auth{}
	r.cache = ttlcache.New[string, *protobuf.Ipv6AuthInfo](
		ttlcache.WithTTL[string, *protobuf.Ipv6AuthInfo](AuthTtl*time.Second),
		ttlcache.WithDisableTouchOnHit[string, *protobuf.Ipv6AuthInfo](), // 不自动延期
	)
	go r.cache.Start() //开始自动删除过期项目
	return r
}

func (a *Ipv4Auth) Valid(ctx context.Context, username, password, ip string) (exitIp string, resErr error) {
	var authInfo *protobuf.AuthInfo
	storeKey := fmt.Sprint(username, ":Auth")
	value := a.cache.Get(storeKey)
	isTouch := true
	if value != nil {
		authInfo = value.Value()
		isTouch = false
	}

	if authInfo == nil {
		str, err := common.GetRedisDB().Get(ctx, storeKey).Result()
		if err != nil {
			resErr = err
		} else {
			authInfo = &protobuf.AuthInfo{}
			err := json.Unmarshal([]byte(str), authInfo)
			if err != nil {
				resErr = err
			}
		}

	}

	if resErr != nil {
		return "", resErr
	}

	if authInfo != nil {
		if authInfo.Username != username || authInfo.Password != password {
			return "", errors.New("密码错误")
		}

		if _, ok := authInfo.Ips[ip]; !ok {
			return "", errors.New("ip地址错误")
		}

		if isTouch {
			a.cache.Set(storeKey, authInfo, ttlcache.DefaultTTL)
		}
	}
	return ip, nil
}

func (a *Ipv6Auth) Valid(ctx context.Context, username, password, ip string) (exitIp string, resErr error) {
	authInfo := &protobuf.Ipv6AuthInfo{}
	storeKey := fmt.Sprint(username, ":Auth")
	value := a.cache.Get(storeKey)
	isTouch := true
	if value != nil {
		authInfo = value.Value()
		isTouch = false
	}

	if authInfo == nil {
		str, err := common.GetRedisDB().Get(ctx, storeKey).Result()
		if err != nil {
			resErr = err

		} else {
			authInfo = &protobuf.Ipv6AuthInfo{}
			err := proto.Unmarshal([]byte(str), authInfo)
			if err != nil {
				resErr = err
			}
		}

	}

	if resErr != nil {
		return "", resErr
	}

	if authInfo != nil {

		if authInfo.Username != username || authInfo.Password != password {
			return "", errors.New("密码错误")
		}

		if authInfo.Ip == "" {
			return "", errors.New("ip地址不能为空")
		}
		if authInfo.Banned == true {
			return "", errors.New("账号已被封禁")
		}
		if isTouch {
			a.cache.Set(storeKey, authInfo, ttlcache.DefaultTTL)
		}
	}

	return authInfo.Ip, nil
}

//func (m *manager) Valid(ctx context.Context, username, password, ip string) (authInfo *protobuf.AuthInfo, resErr error) {
//	// 创建管道
//	pipe := common.GetRedisDB().Pipeline()
//
//	// 定义要操作的键和元素
//	strKey := fmt.Sprintf("%s_%s", REDIS_AUTH_USERDATA, username)
//	setKey := fmt.Sprintf("%s_%s", REDIS_USER_IPSET, username)
//
//	// 添加获取字符串值和判断集合元素是否存在的操作到管道
//	getOp := pipe.Get(ctx, strKey)
//	sismemberOp := pipe.SIsMember(ctx, setKey, ip)
//
//	// 执行管道操作
//	_, err := pipe.Exec(ctx)
//	if err != nil {
//		resErr = fmt.Errorf("redis管道命令执行失败 error:%+v", err)
//		return
//	}
//
//	// 处理获取字符串值的结果
//	strVal, err := getOp.Result()
//	if err != nil {
//		if err == redis.Nil {
//			resErr = fmt.Errorf("%s用户数据不存在", username)
//			return
//		} else {
//			resErr = fmt.Errorf("获取%s用户数据失败 error:%+v", username, err)
//			return
//		}
//	}
//
//	authInfo = &protobuf.AuthInfo{}
//	err = json.Unmarshal([]byte(strVal), authInfo)
//	if err != nil {
//		resErr = fmt.Errorf("json.Unmarshal解析%s用户数据%s失败 error:%+v", username, strVal, err)
//		return
//	}
//
//	if authInfo.Username != username || authInfo.Password != password {
//		resErr = fmt.Errorf("%s用户密码错误 用户数据%+v", username, authInfo)
//		return
//	}
//
//	// 处理判断集合元素是否存在的结果
//	exists, err := sismemberOp.Result()
//	if err != nil {
//		resErr = fmt.Errorf("检测%s用户ip:%+v 执行命令失败 error:%+v", username, ip, err)
//		return
//	}
//
//	if !exists {
//		resErr = fmt.Errorf("检测%s用户ip:%+v不存在", username, ip)
//		return
//	}
//
//	return authInfo, nil
//}

func (a *Ipv4Auth) ValidShadowSocks(ipAddr string) (password string, resErr error) {
	storeKey := fmt.Sprint(ipAddr, ":ShadowSocks")
	value := a.shadowSocksCache.Get(storeKey)
	isTouch := true
	if value != nil {
		password = value.Value()
		isTouch = false
	}

	if password == "" {
		pwd, err := common.GetRedisDB().Get(context.Background(), storeKey).Result()
		if err != nil {
			resErr = err
		} else {
			password = pwd
		}

	}

	if resErr != nil {
		return "", resErr
	}

	if password != "" {

		if isTouch {
			a.shadowSocksCache.Set(storeKey, password, ttlcache.DefaultTTL)
		}
	}
	return
}

func (a *Ipv6Auth) ValidShadowSocks(ipAddr string) (password string, resErr error) {
	return
}
