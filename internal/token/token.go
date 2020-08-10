package token

import (
	"time"

	"github.com/hashicorp/go-hclog"
)

// A Factory returns a token service when called.
type Factory func(hclog.Logger) (Service, error)

// A Callback is registered in init(), and must not attempt to log or
// initialize.  They allow the order in which factories are called to
// be handled in the right order.
type Callback func()

// The Service type defines the required interface for the Token
// Service.  The service must generate tokens, and be able to validate
// them.
type Service interface {
	Generate(Claims, Config) (string, error)
	Validate(string) (Claims, error)
}

// The Config struct contains information that should be used when
// generating a token.
type Config struct {
	Lifetime  time.Duration
	Issuer    string
	IssuedAt  time.Time
	NotBefore time.Time
}

var (
	lb        hclog.Logger
	services  map[string]Factory
	callbacks []Callback

	lifetime time.Duration
)

func init() {
	services = make(map[string]Factory)
}

// New returns an initialized token service based on the value of the
// --token_impl flag.
func New(backend string) (Service, error) {
	t, ok := services[backend]
	if !ok {
		return nil, ErrUnknownTokenService
	}
	return t(log())
}

// Register is called by implementations to register ServiceFactory
// functions.
func Register(name string, impl Factory) {
	if _, ok := services[name]; ok {
		// Already registered
		return
	}
	services[name] = impl
}

// RegisterCallback registers a callback for later execution.
func RegisterCallback(cb Callback) {
	callbacks = append(callbacks, cb)
}

// DoCallbacks executes all callbacks currently registered.
func DoCallbacks() {
	for _, cb := range callbacks {
		cb()
	}
}

// GetConfig returns a struct containing the configuration for the
// token service to use while issuing tokens.
func GetConfig() Config {
	if lifetime == time.Duration(0) {
		lifetime = time.Minute * 5
	}

	return Config{
		Lifetime:  lifetime,
		IssuedAt:  time.Now(),
		NotBefore: time.Now(),
	}
}

// SetParentLogger sets the parent logger for this instance.
func SetParentLogger(l hclog.Logger) {
	lb = l.Named("token")
}

// SetLifetime sets up the lifetime used by tokens that are
// issued later on.
func SetLifetime(t time.Duration) {
	lifetime = t
}

// log is a convenience function that will return a null logger if a
// parent logger has not been specified, mostly useful for tests.
func log() hclog.Logger {
	if lb == nil {
		lb = hclog.NewNullLogger()
	}
	return lb
}
