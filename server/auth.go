package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"proxy_server/config"
	util "proxy_server/utils"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"google.golang.org/protobuf/proto"

	"proxy_server/common"
	"proxy_server/protobuf"
)

const AuthTtl = 30
const DynamicIpv6Prefix = "dynamicIpv6Prefix:"
const CitySegPrefix = "citySegPrefix:"
const AllCitySegSetPrefix = "allCitySeg"

type Auth interface {
	Valid(ctx context.Context, username, password, ip, peerIp string) (exitIp string, resErr error)
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
	go r.cache.Start() // 开始自动删除过期项目
	go r.shadowSocksCache.Start()
	return r
}

func NewIpv6Auth() *Ipv6Auth {
	r := &Ipv6Auth{}
	r.cache = ttlcache.New[string, *protobuf.Ipv6AuthInfo](
		ttlcache.WithTTL[string, *protobuf.Ipv6AuthInfo](AuthTtl*time.Second),
		ttlcache.WithDisableTouchOnHit[string, *protobuf.Ipv6AuthInfo](), // 不自动延期
	)
	go r.cache.Start() // 开始自动删除过期项目
	return r
}

func (a *Ipv4Auth) Valid(ctx context.Context, username, password, ip, peerIp string) (exitIp string, resErr error) {
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

func (a *Ipv6Auth) Valid(ctx context.Context, username, password, ip, peerIp string) (exitIp string, resErr error) {
	if password == "elfproxy_dynamic_ipv6" {
		return validDynamicIpv6(ctx, username, peerIp)
	}
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

func validDynamicIpv6(ctx context.Context, username, peerIp string) (string, error) {
	if !slices.Contains(config.GetConf().Ipv6GatewayIp, peerIp) {
		return "", errors.New("非法的连接，不是动态ipv6 网关 " + peerIp)
	}
	userInfo := strings.Split(username, "_")
	if len(userInfo) < 3 {
		return "", errors.New("动态ipv6 username 信息不足")
	}
	city := userInfo[0]
	session := userInfo[1]
	lifeTime := userInfo[2]
	if session == "" || lifeTime == "" {
		return "", errors.New("动态ipv6 session or lifeTime 为空")
	}
	exitIp, err := common.GetRedisDB().Get(ctx, DynamicIpv6Prefix+session).Result()
	if err == nil && exitIp != "" {
		return exitIp, nil
	}

	var seg string
	if city == "" { //随机一个seg
		seg, err = common.GetRedisDB().SRandMember(ctx, AllCitySegSetPrefix).Result()
		if err != nil {
			return "", errors.New("动态ipv6 Rand city seg err,err:" + err.Error())
		}
	} else {
		seg, err = common.GetRedisDB().Get(ctx, CitySegPrefix+city).Result()
		if err != nil {
			return "", errors.New("动态ipv6 city Seg 获取失败 err:" + err.Error())
		}
	}

	exitIp, err = util.RandomIPv6Address(seg)
	if err != nil {
		return "", errors.New("动态ipv6 cidr解析失败,err:" + err.Error())
	}
	expireTime, err := strconv.Atoi(lifeTime)
	if err != nil {
		return "", errors.New("动态ipv6 lifeTime 格式err:" + err.Error())
	}
	common.GetRedisDB().Set(ctx, DynamicIpv6Prefix+session, exitIp, time.Duration(expireTime)*time.Second)
	return exitIp, nil
}

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
