// Code generated by optiongen. DO NOT EDIT.
// optiongen: github.com/timestee/optiongen

package xfile

import (
	"log"

	"github.com/sandwich-go/xconf"
	"github.com/sandwich-go/xconf-providers/status"
	"github.com/sandwich-go/xconf/kv"
)

// Options should use NewOptions to initialize it
type Options struct {
	KVOption   []kv.Option
	LogDebug   xconf.LogFunc
	LogWarning xconf.LogFunc
	OnUpdate   status.OnConfUpdate
}

// NewOptions new Options
func NewOptions(opts ...Option) *Options {
	cc := newDefaultOptions()
	for _, opt := range opts {
		opt(cc)
	}
	if watchDogOptions != nil {
		watchDogOptions(cc)
	}
	return cc
}

// ApplyOption apply mutiple new option
func (cc *Options) ApplyOption(opts ...Option) {
	for _, opt := range opts {
		opt(cc)
	}
}

// Option option func
type Option func(cc *Options)

// WithKVOption option func for filed KVOption
func WithKVOption(v ...kv.Option) Option {
	return func(cc *Options) {
		cc.KVOption = v
	}
}

// WithLogDebug option func for filed LogDebug
func WithLogDebug(v xconf.LogFunc) Option {
	return func(cc *Options) {
		cc.LogDebug = v
	}
}

// WithLogWarning option func for filed LogWarning
func WithLogWarning(v xconf.LogFunc) Option {
	return func(cc *Options) {
		cc.LogWarning = v
	}
}

// WithOnUpdate option func for filed OnUpdate
func WithOnUpdate(v status.OnConfUpdate) Option {
	return func(cc *Options) {
		cc.OnUpdate = v
	}
}

// InstallOptionsWatchDog the installed func will called when NewOptions  called
func InstallOptionsWatchDog(dog func(cc *Options)) { watchDogOptions = dog }

// watchDogOptions global watch dog
var watchDogOptions func(cc *Options)

// newDefaultOptions new default Options
func newDefaultOptions() *Options {
	cc := &Options{}

	for _, opt := range [...]Option{
		WithKVOption(nil...),
		WithLogDebug(func(s string) { log.Println("[  DEBUG] " + s) }),
		WithLogWarning(func(s string) { log.Println("[WARNING] " + s) }),
		WithOnUpdate(status.LastStatus.UpdateConf),
	} {
		opt(cc)
	}

	return cc
}

// all getter func
func (cc *Options) GetKVOption() []kv.Option         { return cc.KVOption }
func (cc *Options) GetLogDebug() xconf.LogFunc       { return cc.LogDebug }
func (cc *Options) GetLogWarning() xconf.LogFunc     { return cc.LogWarning }
func (cc *Options) GetOnUpdate() status.OnConfUpdate { return cc.OnUpdate }

// OptionsVisitor visitor interface for Options
type OptionsVisitor interface {
	GetKVOption() []kv.Option
	GetLogDebug() xconf.LogFunc
	GetLogWarning() xconf.LogFunc
	GetOnUpdate() status.OnConfUpdate
}

// OptionsInterface visitor + ApplyOption interface for Options
type OptionsInterface interface {
	OptionsVisitor
	ApplyOption(...Option)
}