package raptorq

import (
	"errors"
	"fmt"

	"github.com/nyarime/gofec/raptorq"
)

// Codec wraps the RaptorQ FEC codec with block management.
type Codec struct {
	K           int // source symbols per block
	T           int // symbol size in bytes
	RepairCount int // number of repair symbols to generate
	fec         *raptorq.Codec
}

// NewCodec creates a RaptorQ codec.
// K: source symbols per block, T: symbol size in bytes, repairRatio: fraction of K for repair symbols.
func NewCodec(K, T int, repairRatio float64) *Codec {
	repairCount := int(float64(K) * repairRatio)
	if repairCount < 1 {
		repairCount = 1
	}
	return &Codec{
		K:           K,
		T:           T,
		RepairCount: repairCount,
		fec:         raptorq.New(K, T),
	}
}

// Block is one encoding unit containing symbols with the same BlockID.
type Block struct {
	BlockID  uint16
	DataLen  uint16 // original data length before padding
	Symbols  []raptorq.Symbol
	received int
}

// NewBlock creates a new block from raw data.
// Data will be padded to K*T bytes if needed.
func (c *Codec) NewBlock(blockID uint16, data []byte) *Block {
	// Pad data to K*T bytes
	padded := make([]byte, c.K*c.T)
	dataLen := len(data)
	if dataLen > c.K*c.T {
		dataLen = c.K * c.T
	}
	copy(padded, data[:dataLen])

	// Encode
	symbols := c.fec.Encode(padded, c.RepairCount)

	return &Block{
		BlockID: blockID,
		DataLen: uint16(dataLen),
		Symbols: symbols,
	}
}

// BlockSize returns the block size (K*T).
func (c *Codec) BlockSize() int {
	return c.K * c.T
}

// HeaderSize returns the UDP packet header size.
const HeaderSize = 14

// EncodePacket serializes a symbol into a UDP packet.
// Format:
//
//	[2] BlockID (big endian)
//	[2] K
//	[2] T
//	[2] DataLen
//	[4] ESI (big endian)
//	[N] SymbolData (T bytes)
func EncodePacket(blockID uint16, K, T int, dataLen uint16, esi uint32, symbolData []byte) []byte {
	buf := make([]byte, HeaderSize+len(symbolData))
	buf[0] = byte(blockID >> 8)
	buf[1] = byte(blockID)
	buf[2] = byte(K >> 8)
	buf[3] = byte(K)
	buf[4] = byte(T >> 8)
	buf[5] = byte(T)
	buf[6] = byte(dataLen >> 8)
	buf[7] = byte(dataLen)
	buf[8] = byte(esi >> 24)
	buf[9] = byte(esi >> 16)
	buf[10] = byte(esi >> 8)
	buf[11] = byte(esi)
	copy(buf[HeaderSize:], symbolData)
	return buf
}

// Packet is a decoded incoming packet.
type Packet struct {
	BlockID    uint16
	K          int
	T          int
	ESI        uint32
	DataLen    uint16
	SymbolData []byte
}

// ErrNotEnoughData is returned when a packet is too short.
var ErrNotEnoughData = errors.New("packet too short")

// DecodePacket parses a UDP packet into a Packet.
func DecodePacket(buf []byte) (*Packet, error) {
	if len(buf) < HeaderSize {
		return nil, ErrNotEnoughData
	}
	p := &Packet{
		BlockID: uint16(buf[0])<<8 | uint16(buf[1]),
		K:       int(uint16(buf[2])<<8 | uint16(buf[3])),
		T:       int(uint16(buf[4])<<8 | uint16(buf[5])),
		DataLen: uint16(buf[6])<<8 | uint16(buf[7]),
		ESI:     uint32(buf[8])<<24 | uint32(buf[9])<<16 | uint32(buf[10])<<8 | uint32(buf[11]),
	}
	dataLen := p.T
	if dataLen < 0 {
		return nil, fmt.Errorf("invalid T: %d", p.T)
	}
	if len(buf[HeaderSize:]) < dataLen {
		return nil, fmt.Errorf("symbol data truncated: have %d, want %d", len(buf[HeaderSize:]), dataLen)
	}
	p.SymbolData = make([]byte, dataLen)
	copy(p.SymbolData, buf[HeaderSize:HeaderSize+dataLen])
	return p, nil
}

// AddSymbol adds a received symbol to a block.
// Returns true if the block is complete (has K symbols).
func (c *Codec) AddSymbol(block *Block, esi uint32, data []byte) bool {
	if block == nil {
		return false
	}
	// Check if we already have this symbol
	for _, s := range block.Symbols {
		if s.ESI == esi {
			return false
		}
	}
	block.Symbols = append(block.Symbols, raptorq.Symbol{
		ESI:  esi,
		Data: data,
	})
	block.received++
	return block.received >= c.K
}

// DecodeBlock decodes a completed block back to raw data.
func (c *Codec) DecodeBlock(block *Block) ([]byte, error) {
	if block.received < c.K {
		return nil, fmt.Errorf("need at least %d symbols, have %d", c.K, block.received)
	}
	data, err := c.fec.Decode(block.Symbols, int(block.DataLen))
	if err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}
	return data, nil
}
