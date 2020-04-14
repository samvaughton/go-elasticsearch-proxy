package config

import (
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"net/url"
	"os"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Proxy   ProxyConfig   `yaml:"proxy"`
	Logging LoggingConfig `yaml:"logging"`
}

type ServerConfig struct {
	Address string          `yaml:"address"`
	Tls     ServerTlsConfig `yaml:"tls"`
}

type ServerTlsConfig struct {
	Enabled         bool   `yaml:"enabled"`
	CertificatePath string `yaml:"certificatePath"`
	PrivateKeyPath  string `yaml:"privateKeyPath"`
}

type ProxyConfig struct {
	Elasticsearch ProxyElasticsearchConfig `yaml:"elasticsearch"`
}

type ProxyElasticsearchConfig struct {
	Host   string `yaml:"host"`
	Scheme string `yaml:"scheme"`
}

type LoggingElasticsearchConfig struct {
	Host                  string `yaml:"host"`
	Scheme                string `yaml:"scheme"`
	Index                 string `yaml:"index"`
	Username              string `yaml:"username"`
	Password              string `yaml:"password"`
	LogBufferSize         int    `yaml:"logBufferSize"`
	QueryDebounceDuration string `yaml:"queryDebounceDuration"`
}

type LoggingConfig struct {
	Level         string                     `yaml:"level"`
	Elasticsearch LoggingElasticsearchConfig `yaml:"elasticsearch"`
}

func (s *ServerConfig) IsTlsValid() bool {
	if s.Tls.CertificatePath == "" || s.Tls.PrivateKeyPath == "" || s.Tls.Enabled == false {
		return false
	}

	return true
}

func (es *ProxyElasticsearchConfig) ParseUrl() (*url.URL, error) {
	return url.Parse(es.Scheme + "://" + es.Host)
}

func (es *LoggingElasticsearchConfig) GetUrl() string {
	return es.Scheme + "://" + es.Host
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
