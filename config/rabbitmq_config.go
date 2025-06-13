package config

type rabbitmq_config struct {
	Host              string `mapstructure:"host" json:"host" yaml:"host"`
	Port              int    `mapstructure:"port" json:"port" yaml:"port"`
	User              string `mapstructure:"user" json:"user" yaml:"user"`
	Password          string `mapstructure:"password" json:"password" yaml:"password"`
	VirtualHost       string `mapstructure:"virtualhost" json:"virtualhost" yaml:"virtualhost"`
	BlacklistExchange string `mapstructure:"blacklist_exchange" json:"blacklist_exchange" yaml:"blacklist_exchange"` ///黑名单交换机
	// AuthInfoExchange         string `mapstructure:"auth_info_exchange" json:"auth_info_exchange" yaml:"auth_info_exchange"`                      ///鉴权信息交换机
	BlacklistAccesslogQueue  string `mapstructure:"blacklist_accesslog_queue" json:"blacklist_accesslog_queue" yaml:"blacklist_accesslog_queue"` ///黑名单上报队列
	AccesslogToInfluxDBQueue string `mapstructure:"accesslog_to_influxDB_queue" json:"accesslog_to_influxDB_queue" yaml:"accesslog_to_influxDB_queue"`
}
