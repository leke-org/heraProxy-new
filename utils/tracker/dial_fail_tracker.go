package tracker

import (
	"fmt"
	"github.com/jellydator/ttlcache/v3"
	"log"
	util "proxy_server/utils"
	"sync/atomic"
	"time"
)

const (
	// 配置参数
	windowTime       = 1 * time.Minute  // 计数窗口时间
	threshold        = int32(500)       // 1分钟内最大连接数，使用int32以匹配connectionCount类型
	dialFailCacheTTL = 10 * time.Second //  失败记录缓存过期时间
	blacklistTTL     = 24 * time.Hour   // 黑名单过期时间
)

// DialFailStats 存储每个拨号失败的连接统计信息
type DialFailStats struct {
	failCount     int32 // 连接计数，使用原子操作
	firstConnTime int64 // 第一次连接时间
}

// 添加辅助方法获取时间
func (s *DialFailStats) getFirstConnTime() int64 {
	return atomic.LoadInt64(&s.firstConnTime)
}

// 添加辅助方法设置时间
func (s *DialFailStats) setFirstConnTime(t int64) {
	atomic.StoreInt64(&s.firstConnTime, t)
}

// DialTimeoutTracker - 使用无锁数据结构重构
type DialFailTracker struct {
	// 使用ttlcache进行过期管理
	dialFailCache *ttlcache.Cache[string, *DialFailStats]
	blacklist     *ttlcache.Cache[string, time.Time]
}

func NewDialFailTracker() *DialFailTracker {

	dialFailCache := ttlcache.New[string, *DialFailStats](
		ttlcache.WithTTL[string, *DialFailStats](dialFailCacheTTL),
	)

	blacklist := ttlcache.New[string, time.Time](
		ttlcache.WithTTL[string, time.Time](blacklistTTL),
		ttlcache.WithDisableTouchOnHit[string, time.Time](),
	)

	// 启动过期管理器
	go dialFailCache.Start()
	go blacklist.Start()

	return &DialFailTracker{
		dialFailCache: dialFailCache,
		blacklist:     blacklist,
	}
}

// 检查ip和目标网址是否在黑名单中 - 无锁实现
func (dt *DialFailTracker) IsBlacklisted(ipAndTarget string) bool {
	//if slices.Contains(conf.Conf.DialFailTracker.WhiteList, ipAndTarget) {
	//	return false
	//}
	v := dt.blacklist.Get(ipAndTarget)
	if v != nil {
		return true
	}
	return false
}

// 记录tcp拨号失败连接和对应的目标网址
func (dt *DialFailTracker) RecordDialFailConnection(ipAndTarget string) {

	exist := dt.blacklist.Get(ipAndTarget)
	if exist != nil {
		return
	}
	now := time.Now()

	v, found := dt.dialFailCache.GetOrSet(ipAndTarget, &DialFailStats{
		failCount:     1,
		firstConnTime: now.UnixMilli(),
	})
	if !found { // 说明记录是新建的，不用更新开始计数时间和计数
		return
	}
	if v == nil {
		log.Println("读取DialFailTracker 缓存出错，初始缓存为nil", found)
		return
	}

	dialFailStats := v.Value()
	maxAttempts := 10
	for attempt := 0; attempt < maxAttempts; attempt++ {
		oldFirstConnTime := dialFailStats.getFirstConnTime()
		if now.UnixMilli()-oldFirstConnTime > windowTime.Milliseconds() { // 如果超过了计数窗口，尝试重置计数和开始计数的时间
			if atomic.CompareAndSwapInt64(&dialFailStats.firstConnTime, oldFirstConnTime, now.UnixMilli()) {
				// 成功更新firstConnTime后，重置连接计数
				atomic.StoreInt32(&dialFailStats.failCount, 1)
				break // 成功更新，退出循环
			}
		} else {
			// 不需要重置，只需增加连接计数
			newCount := atomic.AddInt32(&dialFailStats.failCount, 1)
			// 检查是否超过阈值
			if newCount > threshold {
				// 加入黑名单
				dt.blacklist.Set(ipAndTarget, now, blacklistTTL)
				log.Printf("IPAndTarget %s exceeded connection threshold (%d in %v). Adding to blacklist.",
					ipAndTarget, newCount, windowTime)
				util.SendAlarmToDingTalk(fmt.Sprintf("检测到疑似恶意攻击的大量转发请求，使用的ip与目标网址为:[%s]， %d秒内触发dial timeout%d次，已拉入黑名单。",
					ipAndTarget, int(windowTime.Seconds()), newCount))
				return
			}
			break // 不需要重置，退出循环
		}
	}
	return
}

// 关闭并清理资源
func (dt *DialFailTracker) Close() {
	// 停止过期管理器
	dt.dialFailCache.Stop()
	dt.blacklist.Stop()
}
