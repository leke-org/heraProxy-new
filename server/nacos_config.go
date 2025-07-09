package server

import (
	"context"
	"fmt"
	util "proxy_server/utils"
	"time"

	"go.uber.org/zap" // 高性能日志库

	"proxy_server/config"
	"proxy_server/log"

	_ "github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	nacos_remote "github.com/yoyofxteam/nacos-viper-remote"
)

type NacosConfig struct {
	Blacklist     []string
	LimitedReader struct {
		ReadRate  int
		ReadBurst int
	}
	OneIpMaxConn           int
	AlarmOneIPMaxThreshold int
	DialFailTracker        struct {
		WhiteList []string
	}
}

func (m *manager) initNacosConf(groupName string) {
	nacosConfig := &NacosConfig{ //远程如果读取失败，用这个配置
		LimitedReader: struct {
			ReadRate  int
			ReadBurst int
		}{ReadRate: 1000000, ReadBurst: 6000000},
		OneIpMaxConn: 10000,
	}
	m.setNacosConf(nacosConfig)
	successInit := m.tryInitNacosConf(groupName)
	if !successInit {
		util.SendAlarmToDingTalk(fmt.Sprintf("静态ip转发服务器%s，初始化nacos配置失败！请排查原因！", config.GetConf().Nacos.LocalIP))
	}
	// 启动异步初始化
	//go m.initNacosConfAsync(groupName)
}

func (m *manager) initNacosConfAsync(groupName string) {
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	beAlarmed := false

	for attempt := 0; ; attempt++ {
		if m.tryInitNacosConf(groupName) {
			log.Info("[nacos_config] Nacos配置初始化成功", zap.Int("attempt", attempt+1))
			break
		}

		// 计算退避延迟时间（指数退避）
		delay := time.Duration(attempt+1) * baseDelay
		if delay > maxDelay {
			delay = maxDelay
		}

		if !beAlarmed {
			util.SendAlarmToDingTalk("")
			beAlarmed = true
		}

		log.Warn("[nacos_config] Nacos配置初始化失败，准备重试",
			zap.Int("attempt", attempt+1),
			zap.Duration("retry_after", delay))

		time.Sleep(delay)
	}

}

func (m *manager) tryInitNacosConf(groupName string) bool {
	defer func() {
		if r := recover(); r != nil {
			log.Error("[nacos_config] 初始化过程中发生panic", zap.Any("panic", r))
		}
	}()

	m.viperClient = viper.New()

	// 配置 Viper for Nacos 的远程仓库参数
	nacos_remote.SetOptions(&nacos_remote.Option{
		Url:         config.GetConf().Nacos.Url,                                                                                     // nacos server 多地址需要地址用;号隔开，如 Url: "loc1;loc2;loc3"
		Port:        uint64(config.GetConf().Nacos.Port),                                                                            // nacos server端口号
		NamespaceId: config.GetConf().Nacos.NamespaceId,                                                                             // nacos namespace
		GroupName:   groupName,                                                                                                      // nacos group
		Config:      nacos_remote.Config{DataId: config.GetConf().Nacos.LocalIP},                                                    // nacos DataID
		Auth:        &nacos_remote.Auth{Enable: true, User: config.GetConf().Nacos.User, Password: config.GetConf().Nacos.Password}, // 如果需要验证登录,需要此参数
	})

	err := m.viperClient.AddRemoteProvider("nacos", fmt.Sprint(config.GetConf().Nacos.Url, ":", config.GetConf().Nacos.Port), "")
	if err != nil {
		log.Error("[nacos_config] 添加远程提供者失败", zap.Error(err))
		return false
	}

	m.viperClient.SetConfigType("json")
	err = m.viperClient.ReadRemoteConfig()
	if err != nil {
		log.Error("[nacos_config] 读取远程配置失败", zap.Error(err))
		return false
	}

	provider := nacos_remote.NewRemoteProvider("json")
	m.nacosRespChan = provider.WatchRemoteConfigOnChannel(m.viperClient)

	err = m.viperClient.WatchRemoteConfigOnChannel()
	if err != nil {
		log.Error("[nacos_config] WatchRemoteConfigOnChannel失败", zap.Error(err))
		return false
	}
	err = m.viperClient.WatchRemoteConfig()
	if err != nil {
		log.Error("[nacos_config] WatchRemoteConfig失败", zap.Error(err))
		return false
	}

	nacosConfig := &NacosConfig{}
	err = m.viperClient.Unmarshal(nacosConfig)
	if err != nil {
		log.Error("[nacos_config] 解析配置失败", zap.Error(err))
		return false
	}
	m.setNacosConf(nacosConfig)
	log.Info("[nacos_config] 配置加载成功", zap.Any("config", nacosConfig))

	return true
}

func (m *manager) runNacosConfServer(ctx context.Context) {
	loopTime := 60 * time.Second
	ticker := time.NewTicker(loopTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.nacosRespChan:
			nacosConfig := &NacosConfig{}
			if err := m.viperClient.Unmarshal(nacosConfig); err == nil {
				m.setNacosConf(nacosConfig)
				m.updateLimitedReaderAction()
			}
			fmt.Println(nacosConfig)
		case <-ticker.C:
			nacosConfig := &NacosConfig{}
			if err := m.viperClient.Unmarshal(nacosConfig); err == nil {
				m.setNacosConf(nacosConfig)
				m.updateLimitedReaderAction()
			}
			fmt.Println(nacosConfig)
		}
	}
}

func (m *manager) setNacosConf(c *NacosConfig) {
	m.nacosConfigMu.Lock()
	defer m.nacosConfigMu.Unlock()
	m.nacosConfig = c
}

func (m *manager) getNacosConf() *NacosConfig {
	m.nacosConfigMu.RLock()
	defer m.nacosConfigMu.RUnlock()
	return m.nacosConfig
}

func (m *manager) updateLimitedReaderAction() {
	nacosConfig := m.getNacosConf()
	for v := range m.userCtxMap.Iter() {
		v.Val.a.UpdateParameter(nacosConfig.LimitedReader.ReadRate, nacosConfig.LimitedReader.ReadBurst)
	}
}
