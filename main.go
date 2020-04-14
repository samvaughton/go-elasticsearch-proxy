package main

import (
	"elasticsearch-proxy/config"
	"elasticsearch-proxy/elasticsearch"
	"elasticsearch-proxy/proxy"
	"flag"
	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
	"os"
)

var configLocationFlag = flag.String(
	"config",
	"config.yml",
	"Specifies the configuration file location.",
)

func main() {
	// Config Parsing
	flag.Parse()
	cfg, err := config.LoadFromFile(*configLocationFlag)

	if err != nil {
		log.Fatalf(err.Error())
	}

	// Set logging
	log.SetLevelFromString(cfg.Logging.Level)
	log.SetHandler(text.New(os.Stdout))

	elasticsearch.ConfigureEsLogger(cfg)

	proxy.ConfigureAndStartProxyServer(cfg)
}
