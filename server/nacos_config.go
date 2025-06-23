package server

import (
	"context"
	"fmt"
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
		panic(err)
	}

	m.viperClient.SetConfigType("json")
	err = m.viperClient.ReadRemoteConfig()
	if err != nil {
		log.Panic("[nacos_config] 初始化nacos失败", zap.Error(err))
	}
	m.viperClient.SetConfigType("json")
	err = m.viperClient.ReadRemoteConfig()
	if err != nil {
		log.Panic("[nacos_config] 初始化nacos失败", zap.Error(err))
	}

	provider := nacos_remote.NewRemoteProvider("json")
	m.nacosRespChan = provider.WatchRemoteConfigOnChannel(m.viperClient)

	m.viperClient.WatchRemoteConfigOnChannel()
	m.viperClient.WatchRemoteConfig()
	nacosConfig := &NacosConfig{}
	m.viperClient.Unmarshal(nacosConfig)
	m.setNacosConf(nacosConfig)
	fmt.Println(nacosConfig)
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
