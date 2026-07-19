package raptorq

import (
	"fmt"
	"log"

	"1szt/flags"
)

func run() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fmt.Println("========================================")
	fmt.Println("隧道类型 = RaptorQ+FEC")
	fmt.Println("运行模式 =", *flags.Mode)
	fmt.Printf("RaptorQ 参数: K=%d T=%d repair=%.0f%%\n", *flags.RqK, *flags.RqT, *flags.RqRepair*100)
	fmt.Println("========================================")

	switch *flags.Mode {
	case "server":
		log.Printf("[raptorq] 服务端模式: UDP %s -> TCP %s", *flags.Addr, *flags.TcpPort)
		if err := RunServer(*flags.Addr, *flags.TcpPort, *flags.RqK, *flags.RqT, *flags.RqRepair, *flags.Bandwidth); err != nil {
			log.Fatalf("[raptorq] server error: %v", err)
		}
	case "client":
		log.Printf("[raptorq] 客户端模式: TCP %s -> UDP %s  bandwidth=%d bps", *flags.TcpPort, *flags.Addr, *flags.Bandwidth)
		if err := RunClient(*flags.TcpPort, *flags.Addr, *flags.RqK, *flags.RqT, *flags.RqRepair, *flags.Bandwidth); err != nil {
			log.Fatalf("[raptorq] client error: %v", err)
		}
	default:
		log.Fatalf("[raptorq] 未知模式: %s (使用 server 或 client)", *flags.Mode)
	}
}

func init() {
	flags.Register("raptorq", run)
}
