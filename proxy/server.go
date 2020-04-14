package proxy

import (
	"crypto/tls"
	"elasticsearch-proxy/config"
	"elasticsearch-proxy/elasticsearch"
	"github.com/apex/log"
	"github.com/caddyserver/certmagic"
	"net/http"
	"net/http/httputil"
	"os"
	"time"
)

func ConfigureAndStartProxyServer(cfg config.Config) {
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

	proxyContext := NewReverseProxyHandlerContext(targetUrl, reverseProxy, &esQueue)

	mux.HandleFunc("/", NewReverseProxyHandler(proxyContext))

	go esQueue.Start()

	serv := &http.Server{
		Addr:         cfg.Server.Address,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      mux,
	}

	if cfg.Server.Tls.Enabled && cfg.Server.Tls.UseLetsEncrypt {
		//certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
		certmagic.DefaultACME.Agreed = true
		certmagic.DefaultACME.Email = cfg.Server.Tls.Email

		tlsConfig, err := certmagic.TLS([]string{cfg.Server.Host})
		if err != nil {
			panic(err)
		}

		serv.TLSConfig = tlsConfig
	}

	log.Debug("Proxying to " + cfg.Proxy.Elasticsearch.Scheme + "://" + cfg.Proxy.Elasticsearch.Host)

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
