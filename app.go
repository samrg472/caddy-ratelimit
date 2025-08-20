package caddyrl

import "github.com/caddyserver/caddy/v2"

const moduleName = "rate_limit"

func init() {
	caddy.RegisterModule(RateLimitApp{})
}

type RateLimitApp struct {
	Metrics MetricsConfig `json:"metrics"`
}

type MetricsConfig struct {
	IncludeKey bool `json:"include_key,omitempty"`
}

func (RateLimitApp) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  moduleName,
		New: func() caddy.Module { return new(RateLimitApp) },
	}
}

func (s RateLimitApp) Provision(_ caddy.Context) error {
	return nil
}

func (RateLimitApp) Start() error {
	return nil
}

func (RateLimitApp) Stop() error {
	return nil
}

var (
	_ caddy.App         = (*RateLimitApp)(nil)
	_ caddy.Module      = (*RateLimitApp)(nil)
	_ caddy.Provisioner = (*RateLimitApp)(nil)
)
