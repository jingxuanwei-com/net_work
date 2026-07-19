package flags

import (
	"flag"
	"fmt"
	"log"
)

// ============ 隧道选择 ============

// Tunnel 隧道类型: quic / raptorq
var Tunnel = flag.String("tunnel", "raptorq", "隧道类型: quic / raptorq")

// ============ 通用参数 ============

// 转发地址  服务器是提供外部客户端访问的地址  客户端是连接服务器的地址
var Addr = flag.String("addr", "ipv4.hxzmc.top:24690", "隧道协议地址")

// 本地 TCP 端口   服务器情况下是代理这个端口  客户端是本地创建这个端口
var TcpPort = flag.String("port", "127.0.0.1:25565", "本地 TCP 端口")

// Mode 运行模式: server / client
var Mode = flag.String("mode", "client", "运行模式: server / client")

// Bandwidth 目标带宽 (bps)
var Bandwidth = flag.Uint64("bandwidth", 10_000_000, "目标带宽 (bps)，如 10000000 = 10Mbps")

// ============ QUIC 隧道参数 ============

// Congestion 拥塞控制算法
var Congestion = flag.String("congestion", "brutal", "拥塞控制算法: brutal / bbr")

// BBRProfile BBR 配置文件
var BBRProfile = flag.String("bbr-profile", "aggressive", "BBR 配置文件: aggressive / standard / conservative")

// DisableLoss Brutal 模式无视丢包
var DisableLoss = flag.Bool("disable-loss", true, "Brutal 模式无视丢包 (true=无视丢包猛冲)")

// ============ RaptorQ 隧道参数 ============

// RqK RaptorQ 源符号数
var RqK = flag.Int("k", 64, "RaptorQ 源符号数 (K)")

// RqT RaptorQ 符号大小 (建议 1200 接近 MTU)
var RqT = flag.Int("t", 1200, "RaptorQ 符号大小 (T, bytes)")

// RqRepair 修复符号比例
var RqRepair = flag.Float64("repair", 0.2, "修复符号比例 (占K的比例)")

// ---------- 隧道注册与分发 ----------

var tunnels = make(map[string]func())

// Register 注册隧道运行函数
func Register(name string, run func()) {
	tunnels[name] = run
}

// Run 分发到已注册的隧道
func Run() {
	t := *Tunnel
	fn, ok := tunnels[t]
	if !ok {
		log.Fatalf("未知隧道类型: %s (可用: quic / raptorq)", t)
	}
	fmt.Println("隧道:", t)
	fn()
}

func init() {
	flag.Parse()
}
