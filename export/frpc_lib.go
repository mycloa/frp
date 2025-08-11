// Copyright 2024 frp Export Library
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package export

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/fatedier/frp/client"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/util/version"
	"github.com/fatedier/golib/crypto"
	"log"
)

var (
	clientServices = make(map[string]*ClientService)
	clientsMutex   sync.RWMutex
)

// ProxyConfig 代理配置结构
type ProxyConfig struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	LocalIP    string `json:"localIP"`
	LocalPort  int    `json:"localPort"`
	RemotePort int    `json:"remotePort"`
}

// ClientConfig 客户端配置结构
type ClientConfig struct {
	ServerAddr string        `json:"serverAddr"`
	ServerPort int           `json:"serverPort"`
	Token      string        `json:"token,omitempty"`
	User       string        `json:"user,omitempty"`
	Proxies    []ProxyConfig `json:"proxies"`
}

// ClientService 客户端服务包装器
type ClientService struct {
	service *client.Service
	ctx     context.Context
	cancel  context.CancelFunc
}

func init() {
	crypto.DefaultSalt = "frp"
	// 设置必要的环境变量
	os.Setenv("QUIC_GO_DISABLE_RECEIVE_BUFFER_WARNING", "true")
	if os.Getenv("QUIC_GO_DISABLE_ECN") == "" {
		os.Setenv("QUIC_GO_DISABLE_ECN", "true")
	}
}

// GetVersion 获取版本信息
func GetVersion() string {
	return version.Full()
}

// StartClient 启动 frpc 客户端
// clientID: 客户端唯一标识符，用于管理多个客户端实例
// config: 客户端配置
// 返回: 错误信息
func StartClient(clientID string, config ClientConfig) error {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	// 检查客户端是否已存在
	if _, exists := clientServices[clientID]; exists {
		return fmt.Errorf("客户端 %s 已经在运行", clientID)
	}

	// 验证配置
	if config.ServerAddr == "" {
		return fmt.Errorf("服务器地址不能为空")
	}
	if config.ServerPort <= 0 {
		config.ServerPort = 7000
	}
	if len(config.Proxies) == 0 {
		return fmt.Errorf("代理配置不能为空")
	}

	// 构建 frp 配置
	frpConfig, proxyCfgs, err := buildFRPConfig(config)
	if err != nil {
		return fmt.Errorf("构建配置失赅: %w", err)
	}

	// 创建服务选项
	options := &client.ServiceOptions{
		Common:      &frpConfig.ClientCommonConfig,
		ProxyCfgs:   proxyCfgs,
		VisitorCfgs: []v1.VisitorConfigurer{},
	}

	// 验证配置
	warnings, err := validation.ValidateAllClientConfig(options.Common, options.ProxyCfgs, options.VisitorCfgs)
	if err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}
	_ = warnings // 忽略警告

	// 创建客户端服务
	service, err := client.NewService(*options)
	if err != nil {
		return fmt.Errorf("创建客户端服务失败: %w", err)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	clientService := &ClientService{
		service: service,
		ctx:     ctx,
		cancel:  cancel,
	}

	// 启动服务
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[错误] 客户端服务异常: %v", r)
			}
		}()

		if err := service.Run(ctx); err != nil {
			log.Printf("[错误] 客户端服务运行错误: %v", err)
		}
	}()

	// 保存客户端服务
	clientServices[clientID] = clientService

	log.Printf("[信息] frpc 客户端 %s 启动成功，连接服务器: %s:%d", clientID, config.ServerAddr, config.ServerPort)
	return nil
}

// StopClient 停止 frpc 客户端
// clientID: 客户端唯一标识符
// 返回: 错误信息
func StopClient(clientID string) error {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	service, exists := clientServices[clientID]
	if !exists {
		return fmt.Errorf("客户端 %s 不存在或未运行", clientID)
	}

	// 停止服务
	service.cancel()

	// 从映射中删除
	delete(clientServices, clientID)

	log.Printf("[信息] frpc 客户端 %s 已停止", clientID)
	return nil
}

// StopAllClients 停止所有客户端
func StopAllClients() {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	for clientID, service := range clientServices {
		service.cancel()
		log.Printf("[信息] frpc 客户端 %s 已停止", clientID)
	}

	// 清空映射
	clientServices = make(map[string]*ClientService)
}

// IsClientRunning 检查客户端是否正在运行
// clientID: 客户端唯一标识符
// 返回: 是否运行
func IsClientRunning(clientID string) bool {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	_, exists := clientServices[clientID]
	return exists
}

// GetRunningClients 获取所有正在运行的客户端ID列表
// 返回: 客户端ID列表
func GetRunningClients() []string {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	clients := make([]string, 0, len(clientServices))
	for clientID := range clientServices {
		clients = append(clients, clientID)
	}
	return clients
}

// buildFRPConfig 构建 FRP 内部配置
func buildFRPConfig(config ClientConfig) (*v1.ClientConfig, []v1.ProxyConfigurer, error) {
	frpConfig := &v1.ClientConfig{
		ClientCommonConfig: v1.ClientCommonConfig{
			ServerAddr: config.ServerAddr,
			ServerPort: config.ServerPort,
			User:       config.User,
			// 设置默认日志级别
			Log: v1.LogConfig{
				Level: "info",
			},
			// 设置默认传输协议
			Transport: v1.ClientTransportConfig{
				Protocol: "tcp",
			},
		},
	}

	// 设置认证
	if config.Token != "" {
		frpConfig.Auth.Method = "token"
		frpConfig.Auth.Token = config.Token
	} else {
		// 如果没有 token，设置默认的 token 认证
		frpConfig.Auth.Method = "token"
		frpConfig.Auth.Token = ""
	}

	// 构建代理配置
	proxyCfgs := make([]v1.ProxyConfigurer, len(config.Proxies))
	for i, proxy := range config.Proxies {
		switch proxy.Type {
		case "tcp":
			tcpProxy := &v1.TCPProxyConfig{
				ProxyBaseConfig: v1.ProxyBaseConfig{
					Name: proxy.Name,
					Type: "tcp",
					ProxyBackend: v1.ProxyBackend{
						LocalIP:   proxy.LocalIP,
						LocalPort: proxy.LocalPort,
					},
					Transport: v1.ProxyTransport{
						BandwidthLimitMode: "client", // 设置默认带宽限制模式
					},
				},
				RemotePort: proxy.RemotePort,
			}
			proxyCfgs[i] = tcpProxy
		case "udp":
			udpProxy := &v1.UDPProxyConfig{
				ProxyBaseConfig: v1.ProxyBaseConfig{
					Name: proxy.Name,
					Type: "udp",
					ProxyBackend: v1.ProxyBackend{
						LocalIP:   proxy.LocalIP,
						LocalPort: proxy.LocalPort,
					},
					Transport: v1.ProxyTransport{
						BandwidthLimitMode: "client",
					},
				},
				RemotePort: proxy.RemotePort,
			}
			proxyCfgs[i] = udpProxy
		case "http":
			httpProxy := &v1.HTTPProxyConfig{
				ProxyBaseConfig: v1.ProxyBaseConfig{
					Name: proxy.Name,
					Type: "http",
					ProxyBackend: v1.ProxyBackend{
						LocalIP:   proxy.LocalIP,
						LocalPort: proxy.LocalPort,
					},
					Transport: v1.ProxyTransport{
						BandwidthLimitMode: "client",
					},
				},
			}
			proxyCfgs[i] = httpProxy
		case "https":
			httpsProxy := &v1.HTTPSProxyConfig{
				ProxyBaseConfig: v1.ProxyBaseConfig{
					Name: proxy.Name,
					Type: "https",
					ProxyBackend: v1.ProxyBackend{
						LocalIP:   proxy.LocalIP,
						LocalPort: proxy.LocalPort,
					},
					Transport: v1.ProxyTransport{
						BandwidthLimitMode: "client",
					},
				},
			}
			proxyCfgs[i] = httpsProxy
		default:
			return nil, nil, fmt.Errorf("不支持的代理类型: %s", proxy.Type)
		}
	}

	return frpConfig, proxyCfgs, nil
}
