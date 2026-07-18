package net_work

import (
	"flag"
	"fmt"
)

func Run() {
	mode := flag.String("mode", "client", "启动模式: server / client")
	tcpPort := flag.String("port", "127.0.0.1:25565", "本地 TCP 端口")
	quicAddr := flag.String("addr", "127.0.0.1:25588", "QUIC 地址(服务端用 :port 监听, 客户端连远程)")

	flag.Parse()

	fmt.Println("========================================")
	fmt.Println("启动模式 =", *mode)
	fmt.Println("TCP 端口 =", *tcpPort)
	fmt.Println("QUIC 地址 =", *quicAddr)
	fmt.Println("========================================")

	switch *mode {
	case "server":
		// 服务端: 监听 QUIC → 转发到本地 TCP:port
		// addr 示例 ":25588" 表示监听所有网卡的 25588 端口
		RunServer(*quicAddr, *tcpPort)
	case "client":
		// 客户端: 监听本地 TCP:port → 连接远程 QUIC 服务器
		RunClient(*tcpPort, *quicAddr)
	default:
		fmt.Println("未知模式，请使用 server 或 client")
	}
}
