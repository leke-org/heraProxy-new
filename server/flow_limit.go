package server

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// 流量限制相关配置
const LimitshareSize = 8 // 必须为2的n次方，用于分片数量，提升并发性能

// slimit 表示单个流量分片的结构体
type slimit struct {
	sync.RWMutex
	rflow int64 // 该分片剩余流量（字节数）
}

// LimitedReaderAction 流量控制核心逻辑
type LimitedReaderAction struct {
	readRate         int           // 每秒最大持续读取速率（字节/秒）
	readBurst        int           // 每秒最大突发读取量（字节）
	wMutex           sync.RWMutex  // 写锁，保护参数更新
	nowTimeUnix      int64         // 当前时间戳（秒级）
	residReadueBurst int           // 剩余突发流量
	cc               atomic.Uint64 // 计数器，用于分片轮询
	slimitArray      []*slimit     // 流量分片数组，每个分片独立控制流量
	slimitMaxFlow    int64         // 单个分片的最大流量（readBurst / LimitshareSize）
}

// NewLimitedReaderAction 创建流量控制实例
// 参数：
// - readRate: 持续读取速率（字节/秒），默认30KB
// - readBurst: 突发读取量（字节），默认100MB
func NewLimitedReaderAction(readRate, readBurst int) *LimitedReaderAction {
	if readRate <= 0 {
		readRate = 1024 * 30 // 默认30KB/s
	}
	if readBurst <= 0 {
		readBurst = 1024 * 100000 // 默认100MB
	}

	l := &LimitedReaderAction{
		readRate:         readRate,
		readBurst:        readBurst,
		residReadueBurst: readBurst,                         // 初始剩余突发流量等于总量
		nowTimeUnix:      time.Now().Unix(),                 // 初始化时间戳
		slimitMaxFlow:    int64(readBurst / LimitshareSize), // 计算单个分片最大流量
	}

	// 初始化流量分片数组
	for i := 0; i < LimitshareSize; i++ {
		l.slimitArray = append(l.slimitArray, &slimit{rflow: int64(l.slimitMaxFlow)}) // 每个分片初始流量为最大值
	}

	return l
}

// ReadBurst 返回突发流量上限
func (l *LimitedReaderAction) ReadBurst() int {
	return l.readBurst
}

// UpdateParameter 更新流量控制参数
func (l *LimitedReaderAction) UpdateParameter(readRate, readBurst int) {
	l.wMutex.Lock()
	defer l.wMutex.Unlock()

	// 校验参数合法性，设置默认值
	if readRate <= 0 {
		readRate = 1024 * 30
	}
	if readBurst <= 0 {
		readBurst = 1024 * 100000
	}

	// 更新参数
	l.readRate = readRate
	l.readBurst = readBurst
	l.residReadueBurst = readBurst                      // 重置剩余突发流量
	l.slimitMaxFlow = int64(readBurst / LimitshareSize) // 重新计算分片最大流量
}

// GetNowTimeUnix 返回当前时间戳（秒级）
func (l *LimitedReaderAction) GetNowTimeUnix() int64 {
	return l.nowTimeUnix
}

// GetReadSize 计算本次允许读取的字节数
func (l *LimitedReaderAction) GetReadSize(dataLen int) int {
	// 轮询选择分片（利用原子操作和位运算实现无锁取模）
	idx := l.cc.Add(1) % LimitshareSize
	slimitP := l.slimitArray[idx]
	slimitP.Lock() // 锁定当前分片
	defer slimitP.Unlock()

	// 检查分片剩余流量
	if slimitP.rflow <= 0 {
		l.wMutex.Lock() // 加写锁更新全局流量
		now := time.Now().Unix()
		// 时间戳更新，重置全局剩余流量
		if l.nowTimeUnix != now {
			l.nowTimeUnix = now
			l.residReadueBurst = l.readBurst // 重置突发流量
		}

		// 全局剩余突发流量为0，无法读取
		if l.residReadueBurst == 0 {
			l.wMutex.Unlock()
			return 0
		}

		// 分配流量到当前分片
		if l.residReadueBurst >= int(l.slimitMaxFlow) {
			l.residReadueBurst -= int(l.slimitMaxFlow) // 扣除已分配流量
			slimitP.rflow = l.slimitMaxFlow            // 分片流量重置为最大值
		} else {
			slimitP.rflow = int64(l.residReadueBurst) // 剩余流量全部分配
			l.residReadueBurst = 0                    // 全局剩余流量清零
		}
		l.wMutex.Unlock()
	}

	// 计算允许读取的大小（受限于分片剩余流量和读取速率）
	available := int(slimitP.rflow)
	if l.readRate >= available { // 持续速率足够大，直接取分片剩余流量
		if dataLen >= available {
			dataLen = available // 不超过分片剩余流量
			slimitP.rflow = 0   // 分片流量耗尽
			return dataLen
		}
		slimitP.rflow -= int64(dataLen) // 扣除已读取流量
		return dataLen
	}

	// 持续速率限制读取量
	if l.readRate >= dataLen { // 允许读取请求量
		slimitP.rflow -= int64(dataLen)
		return dataLen
	}
	// 只能读取持续速率允许的量
	slimitP.rflow -= int64(l.readRate)
	return l.readRate
}

// 流量限制读取器
type LimitedReader struct {
	r             io.Reader            // 底层读取器
	buf           []byte               // 缓冲区
	llen          int                  // 缓冲区剩余数据长度
	action        *LimitedReaderAction // 流量控制实例
	ctx           context.Context      // 上下文（用于超时控制）
	shortDuration time.Duration        // 短时等待时长（33ms）
	tempbuf       []byte               // 临时缓冲区（2KB）
}

// NewLimitedReader 创建带流量限制的读取器
func NewLimitedReader(ctx context.Context, r io.Reader, action *LimitedReaderAction) *LimitedReader {
	l := &LimitedReader{
		ctx:           ctx,
		r:             r,
		llen:          0,
		action:        action,
		tempbuf:       make([]byte, 2*1024),  // 2KB临时缓冲区
		shortDuration: 33 * time.Millisecond, // 33ms等待周期（约30次/秒）
	}

	l.buf = l.tempbuf // 初始缓冲区指向临时缓冲区
	return l
}

// Read 实现io.Reader接口，带流量限制的读取方法
func (l *LimitedReader) Read(p []byte) (n int, err error) {
	select {
	case <-l.ctx.Done():
		return 0, l.ctx.Err()
	default:
	}

	// 缓冲区无数据时，从底层读取数据到缓冲区
	if l.llen == 0 {
		l.buf = l.tempbuf             // 重置缓冲区为临时缓冲区
		l.llen, err = l.r.Read(l.buf) // 读取数据到缓冲区
		if err != nil {
			return 0, err // 底层读取错误直接返回
		}
	}

	// 缓冲区无有效数据，返回EOF
	if l.llen <= 0 {
		return 0, io.EOF
	}

	// 获取允许读取的字节数（受流量限制）
	size := l.action.GetReadSize(l.llen)
	if size <= 0 { // 无可用流量时等待
		// 创建短时超时上下文
		ctx, cancel := context.WithTimeout(l.ctx, l.shortDuration)
		defer cancel()
		<-ctx.Done() // 等待超时或上下文取消
		size = 1     // 强制读取1字节，避免死锁
	}

	// 限制读取大小不超过缓冲区剩余数据和请求长度
	if len(p) > size {
		p = p[:size] // 截断目标切片
	}

	// 从缓冲区复制数据到目标切片
	copy(p, l.buf[:len(p)])
	l.llen -= len(p)       // 更新缓冲区剩余长度
	l.buf = l.buf[len(p):] // 移动缓冲区指针
	return len(p), nil
}
