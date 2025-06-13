package config

type confData struct {
	Ipv4TcpListenerAddress []string         `mapstructure:"ipv4_tcp_listener_address" json:"ipv4_tcp_listener_address" yaml:"ipv4_tcp_listener_address"` /// [":2423",":5467"]
	Ipv6TcpListenerAddress []string         `mapstructure:"ipv6_tcp_listener_address" json:"ipv6_tcp_listener_address" yaml:"ipv6_tcp_listener_address"` /// [":2423",":5467"]
	Ipv6GatewayIp          []string         `mapstructure:"ipv6_gateway_ip" json:"ipv6_gateway_ip" yaml:"ipv6_gateway_ip"`
	GrpcListenerAddress    string           `mapstructure:"grpc_listenerAddress" json:"grpc_listenerAddress" yaml:"grpc_listenerAddress"`
	LogDir                 string           `mapstructure:"log_dir" json:"log_dir" yaml:"log_dir"`
	LocalIp                string           `mapstructure:"local_ip" json:"local_ip" yaml:"local_ip"`
	ProcessName            string           `mapstructure:"process_name" json:"process_name" yaml:"process_name"`
	Redis                  *redis_config    `mapstructure:"redis" json:"redis" yaml:"redis"`
	Rabbitmq               *rabbitmq_config `mapstructure:"rabbitmq" json:"rabbitmq" yaml:"rabbitmq"`
	Nacos                  *nacos_config    `mapstructure:"nacos" json:"nacos" yaml:"nacos"`
	OpenShadowSocks        bool             `mapstructure:"open_shadowsocks" json:"open_shadowsocks" yaml:"open_shadowsocks"`
	ShadowSocksAddress     string           `mapstructure:"shadowsocks_address" json:"shadowsocks_address" yaml:"shadowsocks_address"`
}
