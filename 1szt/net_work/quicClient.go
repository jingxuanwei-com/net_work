package net_work

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
)

// 作用 连接远程端口quic:ip:port 转换tcp:port
// RunClient 启动 QUIC 代理客户端
//   - localAddr: 本地 TCP 监听地址，如 ":25565"
//   - serverAddr: 远程 QUIC 服务端地址，如 "1.2.3.4:25588"
//
// 工作流程:
//
//	本地 TCP 连接 -> 打开 QUIC 流 -> 发送到远程服务端 :25588 -> 服务端转发到 TCP :25565
func RunClient(localAddr, serverAddr string) {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-proxy"},
	}

	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Fatalf("[客户端] TCP 监听失败: %v", err)
	}
	defer listener.Close()
	log.Printf("[客户端] TCP 代理已启动，监听 %s，QUIC 服务器 %s", localAddr, serverAddr)

	// 客户端连接管理器：自动重连
	cm := newConnManager(serverAddr, tlsConf)
	go cm.keepAlive()

	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			log.Printf("[客户端] 接受 TCP 连接失败: %v", err)
			continue
		}
		go handleClientConn(tcpConn, cm)
	}
}

func handleClientConn(tcpConn net.Conn, cm *connManager) {
	defer tcpConn.Close()

	qConn := cm.get()
	if qConn == nil {
		log.Printf("[客户端] 无可用 QUIC 连接，丢弃 TCP 连接 %s", tcpConn.RemoteAddr())
		return
	}

	stream, err := qConn.OpenStreamSync(context.Background())
	if err != nil {
		log.Printf("[客户端] 打开 QUIC 流失败: %v", err)
		return
	}
	defer stream.Close()

	log.Printf("[客户端] 转发: TCP %s <-> QUIC 流", tcpConn.RemoteAddr())
	pipeBidirectional(tcpConn, newStreamConn(stream))
}

// connManager QUIC 连接管理器（支持自动重连）
type connManager struct {
	mu      sync.RWMutex
	conn    *quic.Conn
	addr    string
	tlsConf *tls.Config
}

func newConnManager(addr string, tlsConf *tls.Config) *connManager {
	return &connManager{addr: addr, tlsConf: tlsConf}
}

func (cm *connManager) get() *quic.Conn {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.conn
}

func (cm *connManager) keepAlive() {
	for {
		func() {
			c, err := quic.DialAddr(context.Background(), cm.addr, cm.tlsConf, nil)
			if err != nil {
				log.Printf("[客户端] 连接 QUIC 服务器失败，5秒后重试: %v", err)
				return
			}
			cm.mu.Lock()
			if cm.conn != nil {
				cm.conn.CloseWithError(0, "替换旧连接")
			}
			cm.conn = c
			cm.mu.Unlock()
			log.Printf("[客户端] 已连接到 QUIC 服务器 %s", cm.addr)

			// 阻塞等待连接断开
			<-c.Context().Done()
			log.Printf("[客户端] QUIC 连接已断开，准备重连...")
		}()
		time.Sleep(5 * time.Second)
	}
}
