package lua

import (
	"context"
	"sync"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"proxy_server/common"
	"proxy_server/log"
)

const SetUserDataLuaScript = `
local stringKey = KEYS[1]
local setKey = KEYS[2]

local stringValue = ARGV[1]
local setMembers = {}
for i = 2, #ARGV do
    setMembers[i - 1] = ARGV[i]
end

redis.call('SET', stringKey, stringValue)
redis.call('DEL', setKey)
redis.call('SADD', setKey, unpack(setMembers))

return 1
`

// var SetUserDataLuaScriptShaCode string

// func init() {
// 	// 加载 Lua 脚本
// 	script := redis.NewScript(SetUserDataLuaScript)
// 	sha, err := script.Load(context.Background(), common.GetRedisDB()).Result()
// 	if err != nil {
// 		log.Panic("[lua] 加载SetUserDataLuaScript失败", zap.Error(err))
// 	}
// 	SetUserDataLuaScriptShaCode = sha
// }

var SetUserDataLuaScriptShaCode = sync.OnceValue[string](func() string {
	// 加载 Lua 脚本
	script := redis.NewScript(SetUserDataLuaScript)
	sha, err := script.Load(context.Background(), common.GetRedisDB()).Result()
	if err != nil {
		log.Panic("[lua] 加载SetUserDataLuaScript失败", zap.Error(err))
	}
	return sha
})
