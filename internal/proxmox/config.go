package proxmox

import (
	"fmt"
	"time"
)

type Config struct {
	Endpoints []string
	Auth      AuthConfig
	TLS       TLSConfig
	Timeout   time.Duration
}

type AuthConfig struct {
	Method   string
	APIToken string
	Username string
	Password string
	Realm    string
}

type TLSConfig struct {
	InsecureSkipVerify bool
}

func NewConfig() *Config {
	return &Config{
		Timeout: 30 * time.Second,
		TLS: TLSConfig{
			InsecureSkipVerify: false,
		},
		Auth: AuthConfig{
			Method: "token",
			Realm:  "pam",
		},
	}
}

func (c *Config) validate() error {
	if len(c.Endpoints) == 0 {
		return fmt.Errorf("no endpoints specified")
	}

	switch c.Auth.Method {
	case "token":
		if c.Auth.APIToken == "" {
			return fmt.Errorf("api_token is required when method is 'token'")
		}
	case "password":
		if c.Auth.Username == "" || c.Auth.Password == "" {
			return fmt.Errorf("username and password are required when method is 'password'")
		}
	default:
		return fmt.Errorf("unsupported auth method: %s", c.Auth.Method)
	}

	return nil
}
