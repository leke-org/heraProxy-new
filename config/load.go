package config

import (
	"fmt"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var confDataInstancePointer atomic.Pointer[confData]

func GetConf() *confData {
	return confDataInstancePointer.Load()
}

func Load(path string) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	v.WatchConfig()

	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("config file changed:", e.Name)
		var confDataT confData
		if err = v.Unmarshal(&confDataT); err != nil {
			fmt.Println(err)
		}
		confDataInstancePointer.Store(&confDataT)
	})

	var confDataT confData
	if err = v.Unmarshal(&confDataT); err != nil {
		fmt.Println(err)
	}

	confDataInstancePointer.Store(&confDataT)
}
