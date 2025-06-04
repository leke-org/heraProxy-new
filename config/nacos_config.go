package config

type nacos_config struct {
	Url           string `mapstructure:"url" json:"url" yaml:"url"`
	Port          int    `mapstructure:"port" json:"port" yaml:"port"`
	NamespaceId   string `mapstructure:"namespace_id" json:"namespace_id" yaml:"namespace_id"`
	Ipv4GroupName string `mapstructure:"ipv4_group_name" json:"ipv4_group_name" yaml:"ipv4_group_name"`
	Ipv6GroupName string `mapstructure:"ipv6_group_name" json:"ipv6_group_name" yaml:"ipv6_group_name"`
	User          string `mapstructure:"user" json:"user" yaml:"user"`
	Password      string `mapstructure:"password" json:"password" yaml:"password"`
	LocalIP       string `mapstructure:"local_ip" json:"local_ip" yaml:"local_ip"`
}
