package congestion

import (
	"fmt"
	"log"
	"strings"

	"1szt/quic/congestion/bbr"
	"1szt/quic/congestion/brutal"

	"github.com/apernet/quic-go"
)

const (
	TypeBBR    = "bbr"
	TypeBrutal = "brutal"
	TypeReno   = "reno"
)

// NormalizeType 标准化拥塞控制类型，空字符串默认返回 brutal
func NormalizeType(congestionType string) (string, error) {
	switch normalized := strings.ToLower(congestionType); normalized {
	case "", TypeBrutal:
		return TypeBrutal, nil
	case TypeBBR:
		return TypeBBR, nil
	case TypeReno:
		return TypeReno, nil
	default:
		return "", fmt.Errorf("unsupported congestion type %q", congestionType)
	}
}

func NormalizeBBRProfile(profile string) (string, error) {
	normalized, err := bbr.ParseProfile(profile)
	if err != nil {
		return "", err
	}
	return string(normalized), nil
}

func UseBBR(conn *quic.Conn, profile bbr.Profile) {
	conn.SetCongestionControl(bbr.NewBbrSender(
		bbr.DefaultClock{},
		bbr.GetInitialPacketSize(conn.RemoteAddr()),
		profile,
	))
}

func UseBrutal(conn *quic.Conn, tx uint64, disableLossCompensation bool) {
	conn.SetCongestionControl(brutal.NewBrutalSender(tx, disableLossCompensation))
}

// Config 拥塞控制配置
type Config struct {
	Type                    string // "bbr" 或 "brutal"
	BBRProfile              string // "aggressive" / "standard" / "conservative"
	Bandwidth               uint64 // brutal 模式目标带宽 (bps)，默认 10_000_000
	DisableLossCompensation bool   // brutal 模式是否无视丢包
}

// DefaultConfig 默认配置：Brutal 10Mbps + 无视丢包（最暴力效果最好）
func DefaultConfig() Config {
	return Config{
		Type:                    TypeBrutal,
		BBRProfile:              "aggressive",
		Bandwidth:               10_000_000, // 10Mbps
		DisableLossCompensation: true,
	}
}

// Apply 将拥塞控制配置应用到 QUIC 连接
func Apply(conn *quic.Conn, cfg Config) {
	switch cfg.Type {
	case TypeBrutal:
		UseBrutal(conn, cfg.Bandwidth, cfg.DisableLossCompensation)
		log.Printf("[流控] Brutal 模式已应用，目标带宽 %.2f Mbps，无视丢包=%v",
			float64(cfg.Bandwidth)/1_000_000, cfg.DisableLossCompensation)
	case TypeBBR:
		UseBBR(conn, bbr.Profile(cfg.BBRProfile))
		log.Printf("[流控] BBR 模式已应用，Profile=%s", cfg.BBRProfile)
	case TypeReno:
		log.Printf("[流控] Reno 模式（无操作）")
	}
}

func UseConfigured(conn *quic.Conn, congestionType, bbrProfile string) {
	switch congestionType {
	case TypeReno:
		return
	default:
		UseBBR(conn, bbr.Profile(bbrProfile))
	}
}
