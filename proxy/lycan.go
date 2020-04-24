package proxy

import (
	"elasticsearch-proxy/lycan"
	"elasticsearch-proxy/util"
	"fmt"
	"github.com/apex/log"
	"github.com/tidwall/gjson"
	"net"
	"net/http"
	"net/url"
)

func NewLycanReverseProxyHandler(ctx *ReverseProxyHandlerContext) ReverseProxyHandler {
	ctx.Proxy.Transport = &MiddlewareTransport{
		http.DefaultTransport,
		ctx,
		ProcessLycanRequest,
	}

	return NewBasicReverseProxyHandler(ctx)
}

func ProcessLycanRequest(ctx ReverseProxyHandlerContext, req *http.Request, resp *http.Response, decodedRequestBody string, decodedResponseBody string) {
	// Determine request type
	requestedUrl := req.URL.String()
	requestType := DetermineRequestType(req)

	if req.Method == "OPTIONS" {
		return
	}

	if requestType != RequestPriceRequest {
		fields := GenerateDefaultFields(requestType, requestedUrl, req)
		util.LogData(&fields)

		return
	}

	fields := GenerateLycanQueryFields(
		requestType,
		requestedUrl,
		req,
		gjson.Parse(decodedResponseBody),
		resp.StatusCode,
	)

	if ctx.Filters.Process(req, fields) {
		// Since this is the elasticsearch queries, we want to de-bounce which is handled by the queue
		ctx.Queue.Channel <- QueueLogEntry{
			Key:    fmt.Sprintf("%s", fields.Get("ip")),
			Fields: fields,
		}
	} else {
		log.Debug("Request did not match the provided filters")
	}
}

func GenerateLycanQueryFields(requestType int, requestedUrl string, req *http.Request, queryResponse gjson.Result, statusCode int) log.Fields {
	// Since we know this is an ES query, we can extract the index
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
		"index":    req.Header.Get("X-Index"),
		"userAgent": req.Header.Get("User-Agent"),
		"rawParams": req.URL.Query().Encode(),
		"data":     lycan.ExtractPriceRequestData(req.URL.Query(), queryResponse, statusCode),
	}
}
