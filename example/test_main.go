package main

import (
	"fmt"
	"log"
	"time"

	"github.com/fatedier/frp/export"
)

func main() {
	// 创建客户端配置
	config := export.ClientConfig{
		ServerAddr: "127.0.0.1", // 使用本地地址避免网络连接问题
		ServerPort: 7000,
		Token:      "123",
		Proxies: []export.ProxyConfig{
			{
				Name:       "test-tcp",
				Type:       "tcp",
				LocalIP:    "127.0.0.1",
				LocalPort:  12345,
				RemotePort: 12345,
			},
		},
	}

	// 获取版本信息
	fmt.Println("frp 版本:", export.GetVersion())

	// 启动客户端
	clientID := "test-client"
	fmt.Printf("正在启动 frpc 客户端 %s...\n", clientID)

	if err := export.StartClient(clientID, config); err != nil {
		log.Printf("启动客户端失败 (预期的，因为没有 frps 服务器): %v", err)
	} else {
		fmt.Printf("客户端 %s 启动成功！\n", clientID)
	}

	// 检查客户端状态
	if export.IsClientRunning(clientID) {
		fmt.Printf("客户端 %s 正在运行\n", clientID)
		
		// 等待 2 秒
		time.Sleep(2 * time.Second)
		
		// 停止客户端
		if err := export.StopClient(clientID); err != nil {
			log.Printf("停止客户端失败: %v", err)
		} else {
			fmt.Printf("客户端 %s 已成功停止\n", clientID)
		}
	}

	// 列出所有运行的客户端
	runningClients := export.GetRunningClients()
	fmt.Printf("当前运行的客户端: %v\n", runningClients)

	// 测试多客户端功能
	fmt.Println("\n=== 测试多客户端功能 ===")
	multiClientTest()

	fmt.Println("测试完成！")
}

func multiClientTest() {
	configs := map[string]export.ClientConfig{
		"client1": {
			ServerAddr: "127.0.0.1",
			ServerPort: 7000,
			Proxies: []export.ProxyConfig{{
				Name: "tcp1", Type: "tcp",
				LocalIP: "127.0.0.1", LocalPort: 8001, RemotePort: 8001,
			}},
		},
		"client2": {
			ServerAddr: "127.0.0.1",
			ServerPort: 7000,
			Proxies: []export.ProxyConfig{{
				Name: "tcp2", Type: "tcp",
				LocalIP: "127.0.0.1", LocalPort: 8002, RemotePort: 8002,
			}},
		},
	}

	// 尝试启动多个客户端（会失败因为没有服务器，但验证配置没问题）
	for id, config := range configs {
		if err := export.StartClient(id, config); err != nil {
			fmt.Printf("客户端 %s 启动失败 (预期的): %v\n", id, err)
		}
	}

	// 显示运行状态
	running := export.GetRunningClients()
	fmt.Printf("运行中的客户端: %v\n", running)

	// 停止所有客户端
	export.StopAllClients()
	fmt.Println("所有客户端已停止")
}