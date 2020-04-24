package elasticsearch

import (
	"elasticsearch-proxy/config"
	"elasticsearch-proxy/util"
	"github.com/apex/log"
	"github.com/elastic/go-elasticsearch/v7"
)

var EsQueryLogger *log.Logger
var LycanPriceRequestLogger *log.Logger

func ConfigureLoggers(cfg config.Config) {
	esCfg := elasticsearch.Config{
		Username: cfg.Logging.EsCredentials.Username,
		Password: cfg.Logging.EsCredentials.Password,
		Addresses: []string{
			cfg.Logging.EsCredentials.GetUrl(),
		},
	}

	client, err := elasticsearch.NewClient(esCfg)

	if err != nil {
		log.Error(err.Error())
	} else {
		log.Debug(util.LogMsg("Connected to Elasticsearch for query logging on: " + cfg.Logging.EsCredentials.GetUrl()))
	}

	if client == nil {
		panic("Could not configure ES client")
	}

	if EsQueryLogger == nil {
		handler := NewElasticsearchHandler(&ApexHandlerConfig{
			BufferSize: cfg.Logging.ElasticsearchQueries.LogBufferSize,
			IndexName:  cfg.Logging.ElasticsearchQueries.Index,
			Client:     *client,
		})

		EsQueryLogger = &log.Logger{
			Handler: handler,
			Level:   log.InfoLevel,
		}
	}

	if LycanPriceRequestLogger == nil {
		handler := NewElasticsearchHandler(&ApexHandlerConfig{
			BufferSize: cfg.Logging.LycanPriceRequests.LogBufferSize,
			IndexName:  cfg.Logging.LycanPriceRequests.Index,
			Client:     *client,
		})

		LycanPriceRequestLogger = &log.Logger{
			Handler: handler,
			Level:   log.InfoLevel,
		}
	}

}
