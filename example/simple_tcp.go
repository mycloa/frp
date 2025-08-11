package main

import (
	"fmt"
	"log"
	"time"

	"github.com/fatedier/frp/export"
)

func main() {
	// 只使用 TCP 代理的简单配置
	config := export.ClientConfig{
		ServerAddr: "8.219.197.85",
		ServerPort: 7000,
		Token:      "123",
		Proxies: []export.ProxyConfig{
			{
				Name:       "test-tcp",
				Type:       "tcp",
				LocalIP:    "10.0.8.150",
				LocalPort:  80,
				RemotePort: 12345,
			},
		},
	}

	fmt.Println("frp 版本:", export.GetVersion())

	// 启动客户端
	clientID := "simple-client"
	fmt.Printf("启动 TCP 代理客户端 %s...\n", clientID)

	if err := export.StartClient(clientID, config); err != nil {
		log.Fatalf("启动失败: %v", err)
	}

	fmt.Printf("✅ 客户端 %s 启动成功！\n", clientID)
	fmt.Printf("   TCP 代理: 127.0.0.1:12345 -> 服务器:12345\n")

	// 让客户端运行一段时间
	fmt.Println("客户端运行中，等待 50 秒...")
	time.Sleep(50 * time.Second)

	// 停止客户端
	fmt.Println("正在停止客户端...")
	if err := export.StopClient(clientID); err != nil {
		log.Printf("停止失败: %v", err)
	} else {
		fmt.Printf("✅ 客户端 %s 已停止\n", clientID)
	}

	// 验证客户端已停止
	if !export.IsClientRunning(clientID) {
		fmt.Println("✅ 确认客户端已完全停止")
	}

	// 主程序继续运行，不退出
	fmt.Println("\n=== 主程序继续运行中 ===")
	fmt.Println("程序将持续运行，按 Ctrl+C 退出...")

	counter := 0
	for {
		time.Sleep(2 * time.Second)
		counter++
		fmt.Printf("主程序运行中... (%d)\n", counter)

		// 可选：每隔一段时间重新启动客户端进行测试
		if counter%10 == 0 {
			fmt.Println("重新测试启动/停止...")
			testRestart(clientID, config)
		}
	}
}

// 测试重启功能
func testRestart(clientID string, config export.ClientConfig) {
	// 启动
	if err := export.StartClient(clientID, config); err != nil {
		fmt.Printf("重启测试失败: %v\n", err)
		return
	}
	fmt.Println("🔄 测试重启成功")

	// 短暂运行后停止
	time.Sleep(120 * time.Second)
	export.StopClient(clientID)
	fmt.Println("🔄 测试停止成功")
}
