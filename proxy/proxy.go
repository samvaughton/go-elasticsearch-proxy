package proxy

import (
	"elasticsearch-proxy/elasticsearch"
	"fmt"
	"github.com/apex/log"
	"github.com/samvaughton/crawlerdetection"
	"github.com/tidwall/gjson"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

const (
	RequestGeneric = iota
	RequestElasticsearch
)

func GetRequestTypeString(requestType int) string {
	switch requestType {
	case RequestElasticsearch:
		return "ELASTICSEARCH"
	}

	return "GENERIC"
}

type ReverseProxyHandlerContext struct {
	Target  *url.URL
	Proxy   *httputil.ReverseProxy
	Queue   *elasticsearch.Queue
	Filters elasticsearch.FilterProcessor
}

func NewReverseProxyHandlerContext(targetUrl *url.URL, proxy *httputil.ReverseProxy, queue *elasticsearch.Queue) ReverseProxyHandlerContext {
	return ReverseProxyHandlerContext{
		Target:  targetUrl,
		Proxy:   proxy,
		Queue:   queue,
		Filters: elasticsearch.NewFilterProcessor(),
	}
}

type ReverseProxyHandler func(res http.ResponseWriter, req *http.Request)

func NewReverseProxyHandler(ctx ReverseProxyHandlerContext) ReverseProxyHandler {

	// Crawler check
	ctx.Filters.AddFilter(func(req *http.Request, fields log.Fields) bool {
		userAgent := req.Header.Get("User-Agent")

		if userAgent == "" {
			log.WithField("userAgent", userAgent).Debug("No User-Agent provided, skipping")
			return false // Do not accept search req's with no user agent
		}

		if crawlerdetection.IsCrawler(userAgent) {
			log.WithField("userAgent", userAgent).Debug("Crawler detected, skipping")
			return false
		}

		return true
	})

	ctx.Filters.AddFilter(func(req *http.Request, fields log.Fields) bool {
		metrics := fields.Get("data").(map[string]interface{})

		return len(metrics) > 0
	})

	ctx.Filters.AddFilter(func(req *http.Request, fields log.Fields) bool {
		index := fields.Get("index").(string)
		metrics := fields.Get("data").(map[string]interface{})

		if strings.Contains(index, "lovecottages") && len(metrics) == 1 {

			// If there is a single nightly metric that is the default range then we can not log this search request
			if metric, exists := elasticsearch.FindMetricByName(elasticsearch.MetricNightlyLowPrice, metrics); exists {
				nightlyLowMetric := metric.(elasticsearch.MetricRangeData)

				if nightlyLowMetric.Minimum == 0 && nightlyLowMetric.Maximum == 9999 {
					return false
				}
			}

		}

		return true
	})

	return func(res http.ResponseWriter, req *http.Request) {
		//requestPayload := ParseRequestBody(req)

		// Store this before it is set
		requestedUrl := req.URL.String()
		requestType := DetermineRequestType(req)

		// Update the headers to allow for SSL redirection
		req.URL.Host = ctx.Target.Host
		req.URL.Scheme = ctx.Target.Scheme
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Header.Set("X-Forwarded-Origin", req.Header.Get("Origin"))
		req.Host = ctx.Target.Host

		if req.Method != "OPTIONS" {

			if requestType != RequestElasticsearch {
				fields := GenerateDefaultFields(requestType, requestedUrl, req)
				go LogData(&fields)
			} else {

				parsedQueryLines := elasticsearch.ParseQueries(DecodeRequestBodyToString(req))

				for _, query := range elasticsearch.DeDuplicateJsonLines(parsedQueryLines) {
					fields := GenerateElasticsearchQueryFields(requestType, requestedUrl, req, query)

					if ctx.Filters.Process(req, fields) {
						// Since this is the elasticsearch queries, we want to de-bounce which is handled by the queue
						ctx.Queue.Channel <- elasticsearch.QueueLogEntry{
							Key:    fmt.Sprintf("%s", fields.Get("ip")),
							Fields: fields,
						}
					} else {
						log.Debug("Query did not match the provided filters")
					}
				}
			}
		}

		ctx.Proxy.ServeHTTP(res, req)
	}
}

func DetermineRequestType(req *http.Request) int {
	if strings.Contains(req.URL.String(), "/_msearch") || strings.Contains(req.URL.String(), "/_search") {
		return RequestElasticsearch
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

func GenerateElasticsearchQueryFields(requestType int, requestedUrl string, req *http.Request, actualQuery gjson.Result) log.Fields {
	// Since we know this is an ES query, we can extract the index
	indexName := ""
	parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")

	if len(parts) > 0 {
		indexName = parts[0]
	}

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
		"type":     GetRequestTypeString(requestType),
		"url":      requestedUrl,
		"host":     host,
		"app":     	req.Header.Get("X-App"),
		"ip":       ip,
		"index":    indexName,
		"userAgent": req.Header.Get("User-Agent"),
		"rawQuery": actualQuery.String(),
		"data":     elasticsearch.ExtractQueryMetrics(actualQuery),
	}
}

func LogData(fields *log.Fields) {
	log.WithFields(fields).Debug("Generic Request")
}
