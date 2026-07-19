package raptorq

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// RunClient starts the RaptorQ FEC tunnel client.
// Listens on tcpAddr for local connections, encodes data with RaptorQ,
// and sends FEC packets via UDP to serverAddr.
func RunClient(tcpAddr, serverAddr string, K, T int, repairRatio float64, bandwidth uint64) error {
	codec := NewCodec(K, T, repairRatio)

	// Resolve server UDP address
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return fmt.Errorf("resolve UDP %s: %w", serverAddr, err)
	}

	// Listen on local TCP
	tcpListener, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		return fmt.Errorf("listen TCP %s: %w", tcpAddr, err)
	}
	defer tcpListener.Close()

	log.Printf("[raptorq] client listening TCP %s, sending to UDP %s (K=%d T=%d repair=%.1f%% bandwidth=%dbps)",
		tcpAddr, serverAddr, K, T, repairRatio*100, bandwidth)

	for {
		tcpConn, err := tcpListener.Accept()
		if err != nil {
			return fmt.Errorf("accept TCP: %w", err)
		}
		log.Printf("[raptorq] TCP connection from %s", tcpConn.RemoteAddr())

		go func(conn net.Conn) {
			defer conn.Close()
			if err := handleTCPConnection(conn, udpAddr, codec, bandwidth); err != nil {
				log.Printf("[raptorq] handle error: %v", err)
			}
		}(tcpConn)
	}
}

// retBlock tracks incoming return-path blocks from server.
type retBlock struct {
	mu      sync.Mutex
	blocks  map[uint16]*Block
	next    uint16
	pending map[uint16][]byte
}

func handleTCPConnection(tcpConn net.Conn, udpAddr *net.UDPAddr, codec *Codec, bandwidth uint64) error {
	// Create UDP connection for sending/receiving
	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("dial UDP: %w", err)
	}
	defer udpConn.Close()

	// Forward path rate limiter
	pacer := NewPacedSender(bandwidth)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ======== Return path: receive UDP blocks from server, decode, write to TCP ========
	go func() {
		retCodec := NewCodec(codec.K, codec.T, float64(codec.RepairCount)/float64(codec.K))
		retBuf := make([]byte, 65535)
		rb := &retBlock{
			blocks:  make(map[uint16]*Block),
			pending: make(map[uint16][]byte),
		}

		for {
			// Use context cancellation to break out
			udpConn.SetReadDeadline(time.Now().Add(5 * time.Second))
			n, err := udpConn.Read(retBuf)
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					continue
				}
				return
			}

			packet, err := DecodePacket(retBuf[:n])
			if err != nil {
				continue
			}

			rb.mu.Lock()
			block, exists := rb.blocks[packet.BlockID]
			if !exists {
				block = &Block{
					BlockID: packet.BlockID,
					DataLen: packet.DataLen,
				}
				rb.blocks[packet.BlockID] = block
			}

			if retCodec.AddSymbol(block, packet.ESI, packet.SymbolData) {
				// Block complete
				decoded, err := retCodec.DecodeBlock(block)
				if err != nil {
					log.Printf("[raptorq] client decode return block %d failed: %v", block.BlockID, err)
					delete(rb.blocks, packet.BlockID)
					rb.mu.Unlock()
					continue
				}
				delete(rb.blocks, packet.BlockID)
				rb.mu.Unlock()

				// Write to TCP in order
				retWriteOrdered(tcpConn, rb, block.BlockID, decoded)
			} else {
				rb.mu.Unlock()
			}
		}
	}()

	// ======== Forward path: read TCP, encode, send via UDP ========
	var blockID uint32
	fwdBuf := make([]byte, 65535)

	for {
		n, err := tcpConn.Read(fwdBuf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read TCP: %w", err)
		}
		// Send immediately — no batching
		sendBlock(ctx, udpConn, pacer, codec, uint16(atomic.AddUint32(&blockID, 1)-1), fwdBuf[:n])
	}
}

// retWriteOrdered writes decoded return data to TCP in block order.
func retWriteOrdered(tcpConn net.Conn, rb *retBlock, blockID uint16, data []byte) {
	rb.mu.Lock()
	if blockID == rb.next {
		rb.mu.Unlock()
		tcpConn.SetWriteDeadline(time.Now().Add(30 * time.Second))
		tcpConn.Write(data)
		rb.mu.Lock()
		rb.next++
		for {
			pd, ok := rb.pending[rb.next]
			if !ok {
				break
			}
			delete(rb.pending, rb.next)
			rb.mu.Unlock()
			tcpConn.SetWriteDeadline(time.Now().Add(30 * time.Second))
			tcpConn.Write(pd)
			rb.mu.Lock()
			rb.next++
		}
	} else if blockID > rb.next {
		rb.pending[blockID] = data
	}
	rb.mu.Unlock()
}

func sendBlock(ctx context.Context, udpConn *net.UDPConn, pacer *PacedSender, codec *Codec, blockID uint16, data []byte) {
	block := codec.NewBlock(blockID, data)

	for _, sym := range block.Symbols {
		packet := EncodePacket(blockID, codec.K, codec.T, block.DataLen, sym.ESI, sym.Data)

		// Pace the send
		if err := pacer.Wait(ctx, len(packet)); err != nil {
			return
		}

		// Send UDP packet
		if _, err := udpConn.Write(packet); err != nil {
			log.Printf("[raptorq] send error: %v", err)
			return
		}
	}
}
