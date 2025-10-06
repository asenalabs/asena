package balancer

import "github.com/asenalabs/asena/internal/config"

type Balancer interface {
	Next() *config.ServerCfg
}

func New(algorithm string, servers []*config.ServerCfg) Balancer {
	switch algorithm {
	case config.RoundRobin:
		return NewRoundRobin(servers)
	default:
		return NewRoundRobin(servers) // default fallback
	}
}
