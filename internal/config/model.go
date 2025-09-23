package config

import "time"

// ============================== Static ==============================

type AsenaConfig struct {
	Asena          *AsenaCfg          `yaml:"asena,omitempty"`
	Log            *LogCfg            `yaml:"log,omitempty"`
	ProxyTransport *ProxyTransportCfg `yaml:"proxy_transport,omitempty"`
}

type AsenaCfg struct {
	Port        *string
	EnableHTTPS *bool   `yaml:"enable_https,omitempty"`
	TLSCertFile *string `yaml:"tls_cert_file,omitempty"`
	TLSKeyFile  *string `yaml:"tls_key_file,omitempty"`
}

type LogCfg struct {
	Lumberjack *LumberjackCfg `yaml:"lumberjack,omitempty"`
}

type LumberjackCfg struct {
	Path       *string `yaml:"path,omitempty"`
	MaxSize    *int    `yaml:"max_size,omitempty"`
	MaxBackups *int    `yaml:"max_backups,omitempty"`
	MaxAge     *int    `yaml:"max_age,omitempty"`
	Compress   *bool   `yaml:"compress,omitempty"`
}

type ProxyTransportCfg struct {
	DailTimeout           *time.Duration `yaml:"dail_timeout,omitempty"`
	DailKeepalive         *time.Duration `yaml:"dail_keepalive,omitempty"`
	ForceHTTP2            *bool          `yaml:"force_http2,omitempty"`
	MaxIdleConn           *int           `yaml:"max_idle_conn,omitempty"`
	MaxIdleConnPerHost    *int           `yaml:"max_idle_conn_per_host,omitempty"`
	IdleConnTimeout       *time.Duration `yaml:"idle_conn_timeout,omitempty"`
	TLSHandshakeTimeout   *time.Duration `yaml:"tls_handshake_timeout,omitempty"`
	ExpectContinueTimeout *time.Duration `yaml:"expect_continue_timeout,omitempty"`
	TLSMinVersion         *uint16
}
