package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Clients []Client `yaml:"clients"`
	Users   []User   `yaml:"users"`
}

type Client struct {
	ID           string   `yaml:"id"            json:"id"`
	Secret       string   `yaml:"secret"         json:"secret"`
	RedirectURIs []string `yaml:"redirect_uris"  json:"redirect_uris"`
}

type User struct {
	Sub           string   `yaml:"sub"            json:"sub"`
	Username      string   `yaml:"username"       json:"username"`
	Password      string   `yaml:"password"       json:"password,omitempty"`
	Email         string   `yaml:"email"          json:"email"`
	EmailVerified bool     `yaml:"email_verified" json:"email_verified"`
	Name          string   `yaml:"name"           json:"name"`
	Groups        []string `yaml:"groups"         json:"groups"`
	IsAdmin       bool     `yaml:"is_admin"       json:"is_admin"`
}

func Load(filePath string) (*Config, error) {
	if filePath == "" {
		filePath = "./config.yaml"
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}
