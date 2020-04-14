package elasticsearch

import (
	"fmt"
	"github.com/apex/log"
	"github.com/elastic/go-elasticsearch/v7"
	"rentivo-es-proxy/config"
	"time"
)

var EsLogger *log.Logger

func LogMsg(message string) string {
	return fmt.Sprintf("%s ES Logger: %s", time.Now(), message)
}

func ConfigureEsLogger(cfg config.Config) {
	esCfg := elasticsearch.Config{
		Username: cfg.Logging.Elasticsearch.Username,
		Password: cfg.Logging.Elasticsearch.Password,
		Addresses: []string{
			cfg.Logging.Elasticsearch.GetUrl(),
		},
	}

	client, err := elasticsearch.NewClient(esCfg)

	if err != nil {
		log.Error(err.Error())
	} else {
		log.Debug(LogMsg("Connected to Elasticsearch for query logging on: " + cfg.Logging.Elasticsearch.GetUrl()))
	}

	if EsLogger == nil {
		handler := NewElasticsearchHandler(&ApexHandlerConfig{
			BufferSize: cfg.Logging.Elasticsearch.LogBufferSize,
			IndexName:  cfg.Logging.Elasticsearch.Index,
			Client:     *client,
		})

		EsLogger = &log.Logger{
			Handler: handler,
			Level:   log.InfoLevel,
		}
	}

}
