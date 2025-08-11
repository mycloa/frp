package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatedier/frp/export"
)

func main() {
	// 创建客户端配置
	config := export.ClientConfig{
		ServerAddr: "8.219.197.85", // 你的 frps 服务器地址
		ServerPort: 7000,           // 你的 frps 服务器端口
		Token:      "123",          // 如果 frps 设置了 token，请填写
		User:       "",             // 可选，用于区分不同用户的代理
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
	clientID := "my-frpc-client"
	fmt.Printf("正在启动 frpc 客户端 %s...\n", clientID)

	if err := export.StartClient(clientID, config); err != nil {
		log.Fatalf("启动客户端失败: %v", err)
	}

	fmt.Printf("客户端 %s 启动成功！\n", clientID)

	// 检查客户端状态
	if export.IsClientRunning(clientID) {
		fmt.Printf("客户端 %s 正在运行\n", clientID)
	}

	// 列出所有运行的客户端
	runningClients := export.GetRunningClients()
	fmt.Printf("当前运行的客户端: %v\n", runningClients)

	// 等待信号以优雅关闭
	fmt.Println("按 Ctrl+C 停止客户端...")
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// 创建一个定时器来展示客户端运行状态
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-signalChan:
			fmt.Println("\n收到停止信号，正在关闭客户端...")

			// 停止指定客户端
			if err := export.StopClient(clientID); err != nil {
				log.Printf("停止客户端失败: %v", err)
			} else {
				fmt.Printf("客户端 %s 已成功停止\n", clientID)
			}

			// 或者停止所有客户端
			// export.StopAllClients()

			fmt.Println("程序退出")
			return

		case <-ticker.C:
			if export.IsClientRunning(clientID) {
				fmt.Printf("客户端 %s 状态: 运行中\n", clientID)
			} else {
				fmt.Printf("客户端 %s 状态: 已停止\n", clientID)
			}
		}
	}

	for {
		time.Sleep(time.Second)
		fmt.Println("wait")
	}
}

// 演示多客户端使用
func multiClientExample() {
	// 配置 1：TCP 代理
	tcpConfig := export.ClientConfig{
		ServerAddr: "8.219.197.85",
		ServerPort: 7000,
		Proxies: []export.ProxyConfig{
			{
				Name:       "ssh-proxy",
				Type:       "tcp",
				LocalIP:    "127.0.0.1",
				LocalPort:  22,
				RemotePort: 2222,
			},
		},
	}

	// 配置 2：HTTP 代理
	httpConfig := export.ClientConfig{
		ServerAddr: "8.219.197.85",
		ServerPort: 7000,
		Proxies: []export.ProxyConfig{
			{
				Name:      "web-proxy",
				Type:      "http",
				LocalIP:   "127.0.0.1",
				LocalPort: 3000,
			},
		},
	}

	// 启动多个客户端
	clients := []struct {
		id     string
		config export.ClientConfig
	}{
		{"tcp-client", tcpConfig},
		{"http-client", httpConfig},
	}

	for _, client := range clients {
		if err := export.StartClient(client.id, client.config); err != nil {
			log.Printf("启动客户端 %s 失败: %v", client.id, err)
		} else {
			fmt.Printf("客户端 %s 启动成功\n", client.id)
		}
	}

	// 等待 10 秒
	time.Sleep(10 * time.Second)

	// 停止所有客户端
	export.StopAllClients()
	fmt.Println("所有客户端已停止")
}
