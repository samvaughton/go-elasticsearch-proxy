package proxy

import (
	"fmt"
	"github.com/apex/log"
	"github.com/tidwall/gjson"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"rentivo-es-proxy/elasticsearch"
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
	Target *url.URL
	Proxy  *httputil.ReverseProxy
	Queue  *elasticsearch.Queue
}

func NewReverseProxyHandlerContext(targetUrl *url.URL, proxy *httputil.ReverseProxy, queue *elasticsearch.Queue) ReverseProxyHandlerContext {
	return ReverseProxyHandlerContext{
		Target: targetUrl,
		Proxy:  proxy,
		Queue:  queue,
	}
}

type ReverseProxyHandler func(res http.ResponseWriter, req *http.Request)

func NewReverseProxyHandler(ctx ReverseProxyHandlerContext) ReverseProxyHandler {
	return func(res http.ResponseWriter, req *http.Request) {
		//requestPayload := ParseRequestBody(req)

		// Store this before it is set
		requestedUrl := req.URL.String()
		requestType := DetermineRequestType(req)

		// Update the headers to allow for SSL redirection
		req.URL.Host = ctx.Target.Host
		req.URL.Scheme = ctx.Target.Scheme
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Host = ctx.Target.Host

		if req.Method != "OPTIONS" {
			if requestType == RequestElasticsearch {

				if requestType == RequestElasticsearch {
					parsedQueryLines := elasticsearch.ParseQueries(DecodeRequestBodyToString(req))

					for _, query := range elasticsearch.DeDuplicateJsonLines(parsedQueryLines) {
						fields := GenerateElasticsearchQueryFields(requestType, requestedUrl, req, query)

						// Since this is the elasticsearch queries, we want to de-duplicate before we send the data off to elasticsearch
						ctx.Queue.Channel <- elasticsearch.QueueLogEntry{
							Key:    fmt.Sprintf("%s", fields.Get("ip")),
							Fields: fields,
						}
					}
				}

			} else {
				fields := GenerateDefaultFields(requestType, requestedUrl, req)
				go LogData(&fields)
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

	return log.Fields{
		"type": GetRequestTypeString(requestType),
		"url":  requestedUrl,
		"host": req.Header.Get("Host"),
		"ip":   ip,
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

	return log.Fields{
		"type":     GetRequestTypeString(requestType),
		"url":      requestedUrl,
		"host":     req.Header.Get("Host"),
		"ip":       ip,
		"index":    indexName,
		"rawQuery": actualQuery.String(),
		"data":     elasticsearch.ExtractQueryMetrics(actualQuery),
	}
}

func LogData(fields *log.Fields) {
	log.WithFields(fields).Debug("Generic Request")
}
