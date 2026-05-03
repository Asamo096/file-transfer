package xfer

import (
	"time"
)

type EncryptionMode string

const (
	EncryptionRequired EncryptionMode = "required"
	EncryptionOptional EncryptionMode = "optional"
	EncryptionNone     EncryptionMode = "none"
)

type LogLevel string

const (
	LogDebug LogLevel = "debug"
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

type Config struct {
	ListenPort     int            `json:"listen_port"`
	STUNServers    []string       `json:"stun_servers"`
	EnableUPnP     bool           `json:"enable_upnp"`
	EnableDHT      bool           `json:"enable_dht"`
	DHTBootstrap   []string       `json:"dht_bootstrap"`
	MaxFileSize    int64          `json:"max_file_size"`
	ChunkSize      int            `json:"chunk_size"`
	EnableResume   bool           `json:"enable_resume"`
	Encryption     EncryptionMode `json:"encryption"`
	TCPFallback    bool           `json:"tcp_fallback"`
	TCPMasquerade  int            `json:"tcp_masquerade"`
	LogLevel       LogLevel       `json:"log_level"`
	DeviceName     string         `json:"device_name"`
	ReadTimeout    time.Duration  `json:"read_timeout"`
	WriteTimeout   time.Duration  `json:"write_timeout"`
}

func NewDefaultConfig() *Config {
	return &Config{
		ListenPort:    0,
		STUNServers: []string{
			"stun.l.google.com:19302",
			"stun1.l.google.com:19302",
			"stun2.l.google.com:19302",
			"stun3.l.google.com:19302",
			"stun4.l.google.com:19302",
		},
		EnableUPnP:    true,
		EnableDHT:     true,
		DHTBootstrap: []string{
			"bootstrap.libp2p.io:443",
			"ipfs.io:443",
		},
		MaxFileSize:   10 * 1024 * 1024 * 1024,
		ChunkSize:     256 * 1024,
		EnableResume:  true,
		Encryption:    EncryptionRequired,
		TCPFallback:   true,
		TCPMasquerade: 0,
		LogLevel:      LogInfo,
		DeviceName:    "",
		ReadTimeout:   30 * time.Second,
		WriteTimeout:  30 * time.Second,
	}
}