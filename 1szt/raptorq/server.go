package raptorq

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// session tracks a remote client's decoder state and TCP connection.
type session struct {
	clientAddr *net.UDPAddr
	tcpConn    net.Conn
	codec      *Codec
	udpConn    net.PacketConn
	bandwidth  uint64
	blocks     map[uint16]*Block // blockID -> incomplete block
	nextWrite  uint16            // next block ID to write (in-order)
	pending    map[uint16][]byte // blocks decoded but can't write yet
	mu         sync.Mutex
	lastSeen   time.Time
	ctx        context.Context
	cancel     context.CancelFunc
}

type sessionManager struct {
	sessions map[string]*session // remote addr -> session
	mu       sync.RWMutex
}

func newSessionManager() *sessionManager {
	return &sessionManager{
		sessions: make(map[string]*session),
	}
}

func (sm *sessionManager) getOrCreate(addr *net.UDPAddr, codec *Codec, tcpTarget string, udpConn net.PacketConn, bandwidth uint64) *session {
	key := addr.String()
	sm.mu.RLock()
	s, ok := sm.sessions[key]
	sm.mu.RUnlock()
	if ok {
		return s
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()
	// Double-check
	if s, ok := sm.sessions[key]; ok {
		return s
	}

	// Connect to TCP target
	tcpConn, err := net.DialTimeout("tcp", tcpTarget, 10*time.Second)
	if err != nil {
		log.Printf("[raptorq] failed to connect TCP %s: %v", tcpTarget, err)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	s = &session{
		clientAddr: addr,
		tcpConn:    tcpConn,
		codec:      codec,
		udpConn:    udpConn,
		bandwidth:  bandwidth,
		blocks:     make(map[uint16]*Block),
		pending:    make(map[uint16][]byte),
		lastSeen:   time.Now(),
		ctx:        ctx,
		cancel:     cancel,
	}
	sm.sessions[key] = s

	log.Printf("[raptorq] new session from %s -> TCP %s", addr, tcpTarget)

	// Start return path: TCP target -> encode -> UDP back to client
	go s.startReturnPath()

	return s
}

func (sm *sessionManager) remove(key string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if s, ok := sm.sessions[key]; ok {
		s.cancel()
		if s.tcpConn != nil {
			s.tcpConn.Close()
		}
		delete(sm.sessions, key)
	}
}

// RunServer starts the RaptorQ FEC tunnel server.
// Listens on udpAddr for FEC-encoded packets, decodes them,
// and forwards the recovered data to tcpTarget.
func RunServer(udpAddr, tcpTarget string, K, T int, repairRatio float64, bandwidth uint64) error {
	codec := NewCodec(K, T, repairRatio)

	udpConn, err := net.ListenPacket("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("listen UDP %s: %w", udpAddr, err)
	}
	defer udpConn.Close()

	log.Printf("[raptorq] server listening UDP %s, forwarding to TCP %s (K=%d T=%d repair=%.1f%%)",
		udpAddr, tcpTarget, K, T, repairRatio*100)

	sm := newSessionManager()
	go sm.cleanupLoop()

	buf := make([]byte, 65535)
	for {
		n, remoteAddr, err := udpConn.ReadFrom(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			log.Printf("[raptorq] read error: %v", err)
			continue
		}

		packet, err := DecodePacket(buf[:n])
		if err != nil {
			log.Printf("[raptorq] bad packet from %s: %v", remoteAddr, err)
			continue
		}

		remoteUDPAddr, ok := remoteAddr.(*net.UDPAddr)
		if !ok {
			continue
		}

		// Get or create session
		s := sm.getOrCreate(remoteUDPAddr, codec, tcpTarget, udpConn, bandwidth)
		if s == nil {
			continue
		}

		s.mu.Lock()
		s.lastSeen = time.Now()

		// Find or create block
		block, exists := s.blocks[packet.BlockID]
		if !exists {
			block = &Block{
				BlockID: packet.BlockID,
				DataLen: packet.DataLen,
			}
			s.blocks[packet.BlockID] = block
		}

		// Add symbol
		if codec.AddSymbol(block, packet.ESI, packet.SymbolData) {
			// Block is complete, decode it
			decoded, err := codec.DecodeBlock(block)
			if err != nil {
				log.Printf("[raptorq] decode block %d failed: %v", block.BlockID, err)
				delete(s.blocks, packet.BlockID)
				s.mu.Unlock()
				continue
			}

			delete(s.blocks, packet.BlockID)
			s.mu.Unlock()

			// Write to TCP in order
			s.writeOrdered(block.BlockID, decoded)
		} else {
			s.mu.Unlock()
		}
	}
}

// writeOrdered writes decoded data to TCP in block order.
func (s *session) writeOrdered(blockID uint16, data []byte) {
	s.mu.Lock()
	if blockID == s.nextWrite {
		// This is the next block, write it
		s.mu.Unlock()
		s.writeTCP(data)
		s.mu.Lock()
		s.nextWrite++
		// Flush any pending blocks that are now in order
		for {
			pendingData, ok := s.pending[s.nextWrite]
			if !ok {
				break
			}
			delete(s.pending, s.nextWrite)
			s.mu.Unlock()
			s.writeTCP(pendingData)
			s.mu.Lock()
			s.nextWrite++
		}
	} else if blockID > s.nextWrite {
		// Out of order, buffer it
		s.pending[blockID] = data
		s.mu.Unlock()
	} else {
		// Duplicate or already written
		s.mu.Unlock()
	}
}

func (s *session) writeTCP(data []byte) {
	if s.tcpConn == nil {
		return
	}
	// Write with timeout
	s.tcpConn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	_, err := s.tcpConn.Write(data)
	if err != nil {
		log.Printf("[raptorq] TCP write error: %v", err)
	}
}

// cleanupLoop removes stale sessions every 60 seconds.
func (sm *sessionManager) cleanupLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		sm.mu.Lock()
		for key, s := range sm.sessions {
			if time.Since(s.lastSeen) > 120*time.Second {
				s.cancel()
				if s.tcpConn != nil {
					s.tcpConn.Close()
				}
				delete(sm.sessions, key)
				log.Printf("[raptorq] cleaned up stale session %s", key)
			}
		}
		sm.mu.Unlock()
	}
}

// startReturnPath reads from the TCP target, encodes with RaptorQ,
// and sends FEC blocks back to the client via UDP.
func (s *session) startReturnPath() {
	retCodec := NewCodec(s.codec.K, s.codec.T, float64(s.codec.RepairCount)/float64(s.codec.K))
	pacer := NewPacedSender(s.bandwidth)

	buf := make([]byte, 65535)
	var blockID uint32

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		n, err := s.tcpConn.Read(buf)
		if err != nil {
			return
		}

		// Encode and send immediately — no batching
		block := retCodec.NewBlock(uint16(blockID), buf[:n])
		blockID++
		for _, sym := range block.Symbols {
			packet := EncodePacket(block.BlockID, retCodec.K, retCodec.T, block.DataLen, sym.ESI, sym.Data)
			if err := pacer.Wait(s.ctx, len(packet)); err != nil {
				return
			}
			s.udpConn.WriteTo(packet, s.clientAddr)
		}
	}
}
