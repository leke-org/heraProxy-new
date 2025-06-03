package lua

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"proxy_server/common"
	"proxy_server/log"
	"sync"
)

const AddUserDataLuaScript = `
local stringKey = KEYS[1]
local setKey = KEYS[2]

local stringValue = ARGV[1]
local setMembers = {}
for i = 2, #ARGV do
    setMembers[i - 1] = ARGV[i]
end

redis.call('SET', stringKey, stringValue)
redis.call('SADD', setKey, unpack(setMembers))

return 1
`

// var AddUserDataLuaScriptShaCode string

// func init() {
// 	// 加载 Lua 脚本
// 	script := redis.NewScript(AddUserDataLuaScript)
// 	sha, err := script.Load(context.Background(), common.GetRedisDB()).Result()
// 	if err != nil {
// 		log.Panic("[lua] 加载AddUserDataLuaScript失败", zap.Error(err))
// 	}
// 	AddUserDataLuaScriptShaCode = sha
// }


var AddUserDataLuaScriptShaCode = sync. OnceValue[string](func()string{
	// 加载 Lua 脚本
	script := redis.NewScript(AddUserDataLuaScript)
	sha, err := script.Load(context.Background(), common.GetRedisDB()).Result()
	if err != nil {
		log.Panic("[lua] 加载AddUserDataLuaScript失败", zap.Error(err))
	}
	return sha
})  