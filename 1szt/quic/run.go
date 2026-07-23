package quic

import (
	"flag"
	"fmt"

	"1szt/quic/congestion"
)

func Run() {
	mode := flag.String("mode", "client", "启动模式: server / client")
	tcpPort := flag.String("port", "127.0.0.1:25565", "本地 TCP 端口")
	quicAddr := flag.String("addr", "127.0.0.1:24688", "QUIC 地址(服务端用 :port 监听, 客户端连远程)")

	// 流控参数
	congType := flag.String("congestion", "brutal", "拥塞控制算法: brutal / bbr")
	bbrProfile := flag.String("bbr-profile", "aggressive", "BBR 配置文件: aggressive / standard / conservative")
	bandwidth := flag.Uint64("bandwidth", 10_000_000, "Brutal 模式目标带宽 (bps)，如 10000000 = 10Mbps")
	disableLoss := flag.Bool("disable-loss", true, "Brutal 模式无视丢包 (true=无视丢包猛冲)")

	flag.Parse()

	cc := congestion.Config{
		Type:                    *congType,
		BBRProfile:              *bbrProfile,
		Bandwidth:               *bandwidth,
		DisableLossCompensation: *disableLoss,
	}

	fmt.Println("========================================")
	fmt.Println("启动模式 =", *mode)
	fmt.Println("TCP 端口 =", *tcpPort)
	fmt.Println("QUIC 地址 =", *quicAddr)
	fmt.Println("流控算法 =", *congType)
	if *congType == "bbr" {
		fmt.Println("BBR Profile =", *bbrProfile)
	} else {
		fmt.Printf("目标带宽 = %.2f Mbps\n", float64(*bandwidth)/1_000_000)
		fmt.Println("无视丢包 =", *disableLoss)
	}
	fmt.Println("========================================")

	switch *mode {
	case "server":
		RunServer(*quicAddr, *tcpPort, cc)
	case "client":
		RunClient(*tcpPort, *quicAddr, cc)
	default:
		fmt.Println("未知模式，请使用 server 或 client")
	}
}
