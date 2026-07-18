package net_work

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"log"
	"math/big"
	"net"
	"time"

	"github.com/quic-go/quic-go"
)

// 作用 代理端口tcp:ip:port 转换quic:port
// RunServer 启动 QUIC 代理服务端
//   - quicAddr: QUIC 监听地址，如 ":25588"
//   - tcpTarget: 目标 TCP 地址，如 "127.0.0.1:25565"
//
// 工作流程:
//
//	客户端 QUIC 连接 -> 服务端接收 QUIC 流 -> 转发到本地 TCP :25565
func RunServer(quicAddr, tcpTarget string) {
	listener, err := quic.ListenAddr(quicAddr, generateTLSConfig(), nil)
	if err != nil {
		log.Fatalf("[服务端] QUIC 监听失败: %v", err)
	}
	defer listener.Close()
	log.Printf("[服务端] QUIC 代理已启动，监听 %s，转发到 TCP %s", quicAddr, tcpTarget)

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Printf("[服务端] 接受 QUIC 连接失败: %v", err)
			continue
		}
		go handleServerConn(conn, tcpTarget)
	}
}

func handleServerConn(conn *quic.Conn, tcpTarget string) {
	defer conn.CloseWithError(0, "连接关闭")
	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			return
		}
		go pipeStreamToTCP(stream, tcpTarget)
	}
}

func pipeStreamToTCP(stream *quic.Stream, tcpTarget string) {
	defer stream.Close()

	tcpConn, err := net.DialTimeout("tcp", tcpTarget, 10*time.Second)
	if err != nil {
		log.Printf("[服务端] 连接 TCP %s 失败: %v", tcpTarget, err)
		return
	}
	defer tcpConn.Close()

	log.Printf("[服务端] 转发: QUIC 流 <-> TCP %s", tcpTarget)
	pipeBidirectional(newStreamConn(stream), tcpConn)
}

// streamConn 包装 *quic.Stream 实现 net.Conn 接口
type streamConn struct {
	*quic.Stream
	localAddr  net.Addr
	remoteAddr net.Addr
}

func newStreamConn(s *quic.Stream) *streamConn {
	return &streamConn{Stream: s}
}

func (s *streamConn) LocalAddr() net.Addr  { return s.localAddr }
func (s *streamConn) RemoteAddr() net.Addr { return s.remoteAddr }

func pipeBidirectional(a, b net.Conn) {
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(a, b)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(b, a)
		done <- struct{}{}
	}()
	<-done
}

// generateTLSConfig 生成自签名 ECDSA TLS 证书（QUIC 强制要求 TLS）
// ECDSA P-256 比 RSA 2048 密钥更小、握手更快、CPU 开销更低
func generateTLSConfig() *tls.Config {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("生成 ECDSA 密钥失败: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "quic-proxy"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		log.Fatalf("创建证书失败: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		log.Fatalf("编码 ECDSA 私钥失败: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Fatalf("加载 TLS 证书失败: %v", err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-proxy"},
	}
}
