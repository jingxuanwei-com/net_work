package quic

import (
	"fmt"

	"1szt/flags"
	"1szt/quic/congestion"
)

func run() {
	cc := congestion.Config{
		Type:                    *flags.Congestion,
		BBRProfile:              *flags.BBRProfile,
		Bandwidth:               *flags.Bandwidth,
		DisableLossCompensation: *flags.DisableLoss,
	}

	fmt.Println("========================================")
	fmt.Println("启动模式 =", *flags.Mode)
	fmt.Println("TCP 端口 =", *flags.TcpPort)
	fmt.Println("QUIC 地址 =", *flags.Addr)
	fmt.Println("流控算法 =", *flags.Congestion)
	if *flags.Congestion == "bbr" {
		fmt.Println("BBR Profile =", *flags.BBRProfile)
	} else {
		fmt.Printf("目标带宽 = %.2f Mbps\n", float64(*flags.Bandwidth)/1_000_000)
		fmt.Println("无视丢包 =", *flags.DisableLoss)
	}
	fmt.Println("========================================")

	switch *flags.Mode {
	case "server":
		RunServer(*flags.Addr, *flags.TcpPort, cc)
	case "client":
		RunClient(*flags.TcpPort, *flags.Addr, cc)
	default:
		fmt.Println("未知模式，请使用 server 或 client")
	}
}

func init() {
	flags.Register("quic", run)
}
