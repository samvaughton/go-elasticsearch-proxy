package config

import (
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"time"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Proxy   ProxyConfig   `yaml:"proxy"`
	Logging LoggingConfig `yaml:"logging"`
}

type ServerConfig struct {
	Host    string          `yaml:"host"`
	Address string          `yaml:"address"`
	Tls     ServerTlsConfig `yaml:"tls"`
}

type ServerTlsConfig struct {
	Email           string `yaml:"email"`
	Enabled         bool   `yaml:"enabled"`
	UseLetsEncrypt  bool   `yaml:"useLetsEncrypt"`
	CertificatePath string `yaml:"certificatePath"`
	PrivateKeyPath  string `yaml:"privateKeyPath"`
}

type ProxyConfig struct {
	Elasticsearch ProxyHostConfig `yaml:"elasticsearch"`
	Lycan         ProxyHostConfig `yaml:"lycan"`
}

type ProxyHostConfig struct {
	Host   string `yaml:"host"`
	Scheme string `yaml:"scheme"`
}

type Credentials struct {
	Host     string `yaml:"host"`
	Scheme   string `yaml:"scheme"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func (c *Credentials) GetUrl() string {
	return c.Scheme + "://" + c.Host
}

type ElasticsearchIndexQueueConfig struct {
	Index                 string `yaml:"index"`
	LogBufferSize         int    `yaml:"logBufferSize"`
	QueryDebounceDuration string `yaml:"queryDebounceDuration"`
}

func (c *ElasticsearchIndexQueueConfig) ParseDuration() time.Duration {
	duration, err := time.ParseDuration(c.QueryDebounceDuration)

	if err != nil {
		panic("Could not parse query debounce duration: " + c.QueryDebounceDuration)
	}

	return duration
}

type LoggingConfig struct {
	Level                string                        `yaml:"level"`
	EsCredentials        Credentials                   `yaml:"credentials"`
	ElasticsearchQueries ElasticsearchIndexQueueConfig `yaml:"elasticsearchQueries"`
	LycanPriceRequests   ElasticsearchIndexQueueConfig `yaml:"lycanPriceRequests"`
}

func (s *ServerConfig) IsTlsValid() bool {
	if s.Tls.Enabled == false {
		return false
	}

	if s.Tls.UseLetsEncrypt {
		return true
	}

	if s.Tls.CertificatePath == "" || s.Tls.PrivateKeyPath == "" {
		return false // Not using LE, check for paths
	}

	return true
}

func (es *ProxyHostConfig) ParseUrl() *url.URL {
	url, err := url.Parse(es.Scheme + "://" + es.Host)

	if err != nil {
		panic("Could not parse config URL")
	}

	return url
}

func LoadFromFile(name string) (Config, error) {
	file, err := os.Open(name)

	if err != nil {
		return Config{}, err
	}

	defer file.Close()

	return loadFromReader(file)
}

func loadFromReader(reader io.Reader) (Config, error) {
	content, err := ioutil.ReadAll(reader)

	if err != nil {
		return Config{}, err
	}

	config := Config{}

	if err := yaml.Unmarshal(content, &config); err != nil {
		return Config{}, err
	}

	return config, nil
}
