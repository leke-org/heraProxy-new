package config

type redis_config struct {
	Addr         string `mapstructure:"addr" json:"addr" yaml:"addr"`
	Password     string `mapstructure:"password" json:"password" yaml:"password"`
	DB           int    `mapstructure:"db" json:"db" yaml:"db"`
	MinIdleConns int    `mapstructure:"min_idle_conns" json:"min_idle_conns" yaml:"min_idle_conns"` // 最小空闲连接数
	MaxIdleConns int    `mapstructure:"max_idle_conns" json:"max_idle_conns" yaml:"max_idle_conns"` ///最大空闲连接数
	PoolSize     int    `mapstructure:"pool_size" json:"pool_size" yaml:"pool_size"`                // 连接池大小
}
