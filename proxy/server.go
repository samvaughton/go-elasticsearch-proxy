package proxy

import (
	"crypto/tls"
	"elasticsearch-proxy/config"
	"elasticsearch-proxy/elasticsearch"
	"github.com/apex/log"
	"github.com/caddyserver/certmagic"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const (
	RequestGeneric = iota
	RequestElasticsearch
	RequestPriceRequest
)

type ReverseProxyHandlerConfig struct {
	MuxPattern string
	TargetUrl *url.URL
	Queue *Queue
	ProxyHandler func(ctx *ReverseProxyHandlerContext) ReverseProxyHandler
}

type ReverseProxyHandlerContext struct {
	Target  *url.URL
	Proxy   *httputil.ReverseProxy
	Queue   *Queue
	Filters FilterProcessor
}

type ReverseProxyHandler func(res http.ResponseWriter, req *http.Request)

func GetRequestTypeString(requestType int) string {
	switch requestType {
	case RequestElasticsearch:
		return "ELASTICSEARCH"
	case RequestPriceRequest:
		return "PRICE_REQUEST"
	}

	return "GENERIC"
}

func NewSingleHostReverseProxy(targetUrl *url.URL) *httputil.ReverseProxy {
	rp := httputil.NewSingleHostReverseProxy(targetUrl)
	rp.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return rp
}

func NewReverseProxyHandlerContext(targetUrl *url.URL, proxy *httputil.ReverseProxy, queue *Queue) ReverseProxyHandlerContext {
	return ReverseProxyHandlerContext{
		Target:  targetUrl,
		Proxy:   proxy,
		Queue:   queue,
		Filters: NewFilterProcessor(),
	}
}

func ConfigureAndStartProxyServer(cfg config.Config) {
	mux := http.NewServeMux()

	lycanQueue := NewQueue(cfg.Logging.LycanPriceRequests.ParseDuration(), *elasticsearch.LycanPriceRequestLogger)
	esQueue := NewQueue(cfg.Logging.ElasticsearchQueries.ParseDuration(), *elasticsearch.EsQueryLogger)

	handlerConfigs := []ReverseProxyHandlerConfig{
		{
			MuxPattern: "/api/",
			TargetUrl: cfg.Proxy.Lycan.ParseUrl(),
			Queue: &lycanQueue,
			ProxyHandler: NewLycanReverseProxyHandler,
		},
		{
			MuxPattern: "/",
			TargetUrl: cfg.Proxy.Elasticsearch.ParseUrl(),
			Queue: &esQueue,
			ProxyHandler: NewElasticsearchReverseProxyHandler,
		},
	}

	for _, handlerCfg := range handlerConfigs {
		reverseProxy := NewSingleHostReverseProxy(handlerCfg.TargetUrl)
		context := NewReverseProxyHandlerContext(handlerCfg.TargetUrl, reverseProxy, handlerCfg.Queue)

		mux.HandleFunc(handlerCfg.MuxPattern, handlerCfg.ProxyHandler(&context))

		go handlerCfg.Queue.Start()
	}

	serv := &http.Server{
		Addr:         cfg.Server.Address,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      mux,
	}

	if cfg.Server.Tls.Enabled && cfg.Server.Tls.UseLetsEncrypt {
		certmagic.DefaultACME.Agreed = true
		certmagic.DefaultACME.Email = cfg.Server.Tls.Email

		tlsConfig, err := certmagic.TLS([]string{cfg.Server.Host})
		if err != nil {
			panic(err)
		}

		serv.TLSConfig = tlsConfig
	}

	log.Debug("Proxying to " + cfg.Proxy.Elasticsearch.Scheme + "://" + cfg.Proxy.Elasticsearch.Host)

	var err error
	if cfg.Server.IsTlsValid() {
		if cfg.Server.Tls.UseLetsEncrypt {
			log.Debug("Listening on " + cfg.Server.Address + " (with TLS using Let's Encrypt)")
			err = serv.ListenAndServeTLS("", "")
		} else {
			log.Debug("Listening on " + cfg.Server.Address + " (with TLS)")
			err = serv.ListenAndServeTLS(cfg.Server.Tls.CertificatePath, cfg.Server.Tls.PrivateKeyPath)
		}
	} else {
		log.Debug("Listening on " + cfg.Server.Address + " (without TLS)")

		err = serv.ListenAndServe()
	}

	if err != nil {
		log.Fatal(err.Error())
	}
}

func DetermineRequestType(req *http.Request) int {
	if strings.Contains(req.URL.String(), "/_msearch") || strings.Contains(req.URL.String(), "/_search") {
		return RequestElasticsearch
	} else if strings.Contains(req.URL.String(), "/pricing") {
		return RequestPriceRequest
	}

	return RequestGeneric
}

func GenerateDefaultFields(requestType int, requestedUrl string, req *http.Request) log.Fields {
	ip, _, err := net.SplitHostPort(req.RemoteAddr)

	if err != nil {
		log.Error("Failed to split the host/port on remote address: " + req.RemoteAddr)
	}

	parsedOrigin, err := url.Parse(req.Header.Get("Origin"))
	host := ""
	if err == nil {
		host = parsedOrigin.Host
	}

	return log.Fields{
		"type": GetRequestTypeString(requestType),
		"url":  requestedUrl,
		"host": host,
		"app":     	req.Header.Get("X-App"),
		"ip":   ip,
		"userAgent": req.Header.Get("User-Agent"),
		"data": "",
	}
}