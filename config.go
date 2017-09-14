package clickhouse

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// Config is a configuration parsed from a DSN string
type Config struct {
	User         string
	Password     string
	Scheme       string
	Host         string
	Database     string
	Timeout      time.Duration
	IdleTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Location     *time.Location
	Debug        bool
	Params       map[string]string
}

// NewConfig creates a new config with default values
func NewConfig() *Config {
	return &Config{
		Scheme:      "http",
		Host:        "localhost:8123",
		IdleTimeout: time.Hour,
		Location:    time.UTC,
	}
}

// FormatDSN formats the given Config into a DSN string which can be passed to
// the driver.
func (cfg *Config) FormatDSN() string {
	u := cfg.url(nil, true)
	query := u.Query()
	if cfg.Timeout != 0 {
		query.Set("timeout", cfg.Timeout.String())
	}
	if cfg.IdleTimeout != 0 {
		query.Set("idle_timeout", cfg.IdleTimeout.String())
	}
	if cfg.ReadTimeout != 0 {
		query.Set("read_timeout", cfg.ReadTimeout.String())
	}
	if cfg.WriteTimeout != 0 {
		query.Set("write_timeout", cfg.WriteTimeout.String())
	}
	if cfg.Location != time.UTC && cfg.Location != nil {
		query.Set("location", cfg.Location.String())
	}
	if cfg.Debug {
		query.Set("debug", "1")
	}

	u.RawQuery = query.Encode()
	return u.String()
}

func (cfg *Config) url(extra map[string]string, dsn bool) *url.URL {
	u := &url.URL{
		Host:   cfg.Host,
		Scheme: cfg.Scheme,
		Path:   "/",
	}
	if len(cfg.User) > 0 {
		if len(cfg.Password) > 0 {
			u.User = url.UserPassword(cfg.User, cfg.Password)
		} else {
			u.User = url.User(cfg.User)
		}
	}
	query := u.Query()
	if len(cfg.Database) > 0 {
		if dsn {
			u.Path += cfg.Database
		} else {
			query.Set("database", cfg.Database)
		}
	}
	for k, v := range cfg.Params {
		query.Set(k, v)
	}
	if extra != nil {
		for k, v := range extra {
			query.Set(k, v)
		}
	}

	u.RawQuery = query.Encode()
	return u
}

// ParseDSN parses the DSN string to a Config
func ParseDSN(dsn string) (*Config, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	cfg := NewConfig()

	cfg.Scheme = u.Scheme
	cfg.Host = u.Host
	if len(u.Path) > 1 {
		// skip '/'
		cfg.Database = u.Path[1:]
	}
	if u.User != nil {
		// it is expected that empty password will be dropped out on Parse and Format
		cfg.User = u.User.Username()
		if passwd, ok := u.User.Password(); ok {
			cfg.Password = passwd
		}
	}
	if err = parseDSNParams(cfg, map[string][]string(u.Query())); err != nil {
		return nil, err
	}
	return cfg, nil
}

// parseDSNParams parses the DSN "query string"
// Values must be url.QueryEscape'ed
func parseDSNParams(cfg *Config, params map[string][]string) (err error) {
	for k, v := range params {
		if len(v) == 0 {
			continue
		}

		switch k {
		case "timeout":
			cfg.Timeout, err = time.ParseDuration(v[0])
		case "idle_timeout":
			cfg.IdleTimeout, err = time.ParseDuration(v[0])
		case "read_timeout":
			cfg.ReadTimeout, err = time.ParseDuration(v[0])
		case "write_timeout":
			cfg.WriteTimeout, err = time.ParseDuration(v[0])
		case "location":
			cfg.Location, err = time.LoadLocation(v[0])
		case "debug":
			cfg.Debug, err = strconv.ParseBool(v[0])
		case "default_format", "query", "database":
			err = fmt.Errorf("unknown option '%s'", k)
		default:
			// lazy init
			if cfg.Params == nil {
				cfg.Params = make(map[string]string)
			}
			cfg.Params[k] = v[0]
		}
		if err != nil {
			return err
		}
	}

	return
}
