# HeraProxy

HeraProxy 是一个高性能、多协议的代理服务器，支持 IPv4/IPv6 双栈，具有强大的认证、黑名单和流量控制功能。

## 功能特点

- **多协议支持**：HTTP 代理、SOCKS5 代理和 Shadowsocks
- **双栈网络**：同时支持 IPv4 和 IPv6
- **强大的认证系统**：用户名/密码认证，支持 IP 绑定
- **动态 IPv6 分配**：基于城市和会话的动态 IPv6 地址分配
- **安全机制**：黑名单过滤、IP 连接数限制、拨号失败跟踪
- **流量控制**：支持用户级别的流量限制
- **域名嗅探**：从 TLS ClientHello 或 HTTP 请求中提取域名
- **高性能设计**：使用 Go 语言的并发特性，高效处理大量连接
- **可观测性**：详细的日志记录和性能监控

## 系统架构

HeraProxy 由以下核心组件组成：

- **Manager**：核心控制器，协调所有组件
- **TCP 服务器**：处理 TCP 连接的接受和分发
- **代理处理程序**：HTTP、SOCKS5 和 Shadowsocks 协议实现
- **认证系统**：验证用户身份和权限
- **黑名单管理**：过滤不允许的目标地址
- **流量限制**：控制用户的带宽使用

## 依赖服务

HeraProxy 依赖以下外部服务：

- **Redis**：存储用户认证信息、黑名单和动态 IPv6 配置
- **RabbitMQ**：处理黑名单更新、用户数据变更和访问日志
- **Nacos**：服务发现和配置管理

## 安装和构建

### 前提条件

- Go 1.21 或更高版本
- Redis 服务器
- RabbitMQ 服务器
- Nacos 服务器

### 构建

```shell
# 构建ipv4版本
go build -tags ipv4 -o hera_ipv4 main.go http.go

# 构建ipv6版本
go build -tags ipv6 -o hera_ipv6 main.go http.go
```

### 代码格式化

```shell
gofumpt -l -w .
```

## 配置

HeraProxy 使用 YAML 配置文件，默认为 `config.yaml`。主要配置项包括：

```yaml
# 网络配置
listen_addr_ipv4: ":7777"  # IPv4 监听地址
listen_addr_ipv6: []       # IPv6 监听地址列表

# 日志配置
log_dir: "/path/to/logs"   # 日志目录

# 进程配置
local_ip: "x.x.x.x"        # 本地 IP 地址
process_name: "hera_ipv4"  # 进程名称

# Shadowsocks 配置
shadowsocks:
  enable: false            # 是否启用 Shadowsocks
  addr: ":1234"            # Shadowsocks 监听地址

# Redis 配置
redis:
  addr: "host:port"        # Redis 服务器地址
  password: "password"     # Redis 密码
  db: 0                    # Redis 数据库编号

# RabbitMQ 配置
rabbitmq:
  host: "host"             # RabbitMQ 主机
  port: 5672               # RabbitMQ 端口
  username: "username"     # RabbitMQ 用户名
  password: "password"     # RabbitMQ 密码
  vhost: "/"               # RabbitMQ 虚拟主机

# Nacos 配置
nacos:
  url: "host"              # Nacos 服务器地址
  port: 8848               # Nacos 端口
  namespace_id: "id"       # Nacos 命名空间 ID
  group: "group"           # Nacos 组名
  username: "username"     # Nacos 用户名
  password: "password"     # Nacos 密码
```

## 使用方法

### 启动服务

```shell
# 使用默认配置文件
./bin/heraProxy.exe

# 指定配置文件
./bin/heraProxy.exe --config_path=/path/to/config.yaml
```

### HTTP 代理

配置客户端使用 HeraProxy 作为 HTTP 代理：

- **地址**：服务器 IP 地址
- **端口**：配置文件中的 `listen_addr_ipv4` 端口（默认 7777）
- **认证**：用户名和密码

### SOCKS5 代理

配置客户端使用 HeraProxy 作为 SOCKS5 代理：

- **地址**：服务器 IP 地址
- **端口**：配置文件中的 `listen_addr_ipv4` 端口（默认 7777）
- **认证**：用户名和密码

### Shadowsocks

如果启用了 Shadowsocks：

- **地址**：服务器 IP 地址
- **端口**：配置文件中的 `shadowsocks.addr` 端口（默认 1234）
- **加密方法和密码**：根据 Redis 中的配置

## 监控和调试

HeraProxy 提供了 HTTP 端点用于监控和调试：

- `/metrics` - 显示运行时内存统计信息
- `/debug/pprof` - 提供性能分析功能

## 高级功能

### 动态 IPv6 分配

使用特殊格式的用户名和密码可以请求动态 IPv6 地址：

- **用户名格式**：`城市_会话ID_生存时间`
- **密码**：`elfproxy_dynamic_ipv6`

### 黑名单管理

黑名单通过 RabbitMQ 更新，支持以下类型：

- IP 黑名单
- 域名黑名单
- IP-目标组合黑名单

## 许可证

[添加许可证信息]

## 贡献指南

[添加贡献指南