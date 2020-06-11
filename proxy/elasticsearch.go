package proxy

import (
	"elasticsearch-proxy/elasticsearch"
	"elasticsearch-proxy/util"
	"fmt"
	"github.com/apex/log"
	"github.com/samvaughton/crawlerdetection"
	"github.com/tidwall/gjson"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func NewElasticsearchReverseProxyHandler(ctx *ReverseProxyHandlerContext) ReverseProxyHandler {
	ctx.Proxy.Transport = &MiddlewareTransport{
		http.DefaultTransport,
		ctx,
		ProcessElasticRequest,
	}

	// Crawler check
	ctx.Filters.AddFilter(func(req *http.Request, fields log.Fields) bool {
		if req.Header.Get("Debug") != "" {
			return true
		}

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

		if strings.Contains(index, "lovecottages") &&
			(
				len(metrics) == 1 ||
				(len(metrics) == 2 && metrics[elasticsearch.MetricResponse] != nil))  {

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

	return NewBasicReverseProxyHandler(ctx)
}

func ProcessElasticRequest(ctx ReverseProxyHandlerContext, req *http.Request, resp *http.Response, decodedRequestBody string, decodedResponseBody string) {
	// Determine request type
	requestedUrl := req.URL.String()
	requestType := DetermineRequestType(req)

	if req.Method == "OPTIONS" {
		return
	}

	if requestType != RequestElasticsearch {
		fields := GenerateDefaultFields(requestType, requestedUrl, req)
		util.LogData(&fields)

		return
	}

	parsedQueryLines := elasticsearch.ParseQueries(decodedRequestBody)

	deDuplicatedResponseLines := elasticsearch.DeDuplicateJsonLines(gjson.Parse(decodedResponseBody).Get("responses").Array())

	for index, query := range elasticsearch.DeDuplicateJsonLines(parsedQueryLines) {

		var queryResponse gjson.Result
		if len(deDuplicatedResponseLines) > 0 {
			queryResponse = deDuplicatedResponseLines[index]
		}

		fields := GenerateElasticsearchQueryFields(requestType, requestedUrl, req, query, queryResponse)

		if ctx.Filters.Process(req, fields) == false {
			log.Debug("Query did not match the provided filters")

			continue
		}

		// Since this is the elasticsearch queries, we want to de-bounce which is handled by the queue
		ctx.Queue.Channel <- QueueLogEntry{
			Key:    fmt.Sprintf("%s", fields.Get("ip")),
			Fields: fields,
		}
	}
}

func GenerateElasticsearchQueryFields(requestType int, requestedUrl string, req *http.Request, actualQuery gjson.Result, queryResponse gjson.Result) log.Fields {
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
		"data":     elasticsearch.ExtractQueryMetrics(actualQuery, queryResponse),
	}
}