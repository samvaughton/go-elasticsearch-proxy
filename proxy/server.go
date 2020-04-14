package proxy

import (
	"crypto/tls"
	"github.com/apex/log"
	"net/http"
	"net/http/httputil"
	"os"
	"rentivo-es-proxy/config"
	"rentivo-es-proxy/elasticsearch"
	"time"
)

func NewProxyServer(cfg config.Config) *http.Server {
	targetUrl, err := cfg.Proxy.Elasticsearch.ParseUrl()

	if err != nil {
		log.Fatal(err.Error())
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(targetUrl)
	reverseProxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	mux := http.NewServeMux()

	duration, err := time.ParseDuration(cfg.Logging.Elasticsearch.QueryDebounceDuration)
	if err != nil {
		log.Error("Failed to parse query logging debounce duration: " + cfg.Logging.Elasticsearch.QueryDebounceDuration)
		os.Exit(2)
	}

	esQueue := elasticsearch.NewQueue(duration)

	context := NewReverseProxyHandlerContext(targetUrl, reverseProxy, &esQueue)

	mux.HandleFunc("/", NewReverseProxyHandler(context))

	go esQueue.Start()

	serv := &http.Server{
		Addr:    cfg.Server.Address,
		Handler: mux,
	}

	return serv
}

func StartProxyServer(serv *http.Server, cfg config.Config) {
	log.Debug("Proxying to " + cfg.Proxy.Elasticsearch.Scheme + "://" + cfg.Proxy.Elasticsearch.Host)

	var err error

	if cfg.Server.IsTlsValid() {
		log.Debug("Listening on " + cfg.Server.Address + " (with TLS)")

		err = serv.ListenAndServeTLS(cfg.Server.Tls.CertificatePath, cfg.Server.Tls.PrivateKeyPath)
	} else {
		log.Debug("Listening on " + cfg.Server.Address + " (without TLS)")

		err = serv.ListenAndServe()
	}

	log.Fatal(err.Error())

}
