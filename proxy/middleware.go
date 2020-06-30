package proxy

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"elasticsearch-proxy/cache"
	"elasticsearch-proxy/util"
	"fmt"
	"github.com/apex/log"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"
	"time"
)

type MiddlewareTransport struct {
	http.RoundTripper
	Cache *cache.Storage
	ReverseProxyHandlerContext *ReverseProxyHandlerContext
	MiddlewareRoutine func(ctx ReverseProxyHandlerContext, req *http.Request, resp *http.Response, decodedRequestBody string, decodedResponseBody string)
}

func (t *MiddlewareTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	var decodedRequestBody []byte

	if req.URL != nil && req.Header != nil  && req.Body != nil {
		decodedRequestBody = util.DecodeRequestBodyToBytes(req)
	}

	// We need to check the cache now
	// Hash the URL + body
	h := sha1.New()
	h.Write([]byte(req.URL.String()))
	h.Write(decodedRequestBody)
	hash := fmt.Sprintf("%x", h.Sum(nil))

	if t.Cache != nil && t.Cache.Has(hash) {
		cachedBytes := t.Cache.Get(hash)
		cachedResp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(cachedBytes)), req)

		if err != nil {
			log.Errorf("Could not read response for key: " + hash, err)
		}

		cachedResp.Header.Set("X-Cached", hash)

		// we have the cached version, lets return this and start a
		// go routine to finish off the round trip to re-cache the next good response
		// currently commented out as returning the cache response cancels the current http context
		// before it has a chance to execute
		// go t.DoRoundTrip(req, decodedRequestBody, hash)

		// We still want to "log" this request though
		if t.MiddlewareRoutine != nil && len(cachedBytes) > 0 {
			go t.MiddlewareRoutine(
				*t.ReverseProxyHandlerContext,
				req,
				resp,
				string(decodedRequestBody),
				util.DecodeResponseBytes(
					util.DecodeResponseBodyToBytes(cachedResp),
					cachedResp.Header.Get("Content-Encoding"),
				),
			)
		}

		return cachedResp, nil
	}

	return t.DoRoundTrip(req, decodedRequestBody, hash)
}

func (t *MiddlewareTransport) DoRoundTrip(req *http.Request, decodedRequestBody []byte, hash string) (resp *http.Response, err error) {
	// This is where the request is forwarded on to its configured address
	resp, err = t.RoundTripper.RoundTrip(req)

	if err != nil {
		fmt.Println(err)

		return nil, err
	}

	respBytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	// We can modify the response here
	if t.MiddlewareRoutine != nil && len(respBytes) > 0 {
		go t.MiddlewareRoutine(
			*t.ReverseProxyHandlerContext,
			req,
			resp,
			string(decodedRequestBody),
			util.DecodeResponseBytes(
				respBytes,
				resp.Header.Get("Content-Encoding"),
			),
		)
	}

	body := ioutil.NopCloser(bytes.NewReader(respBytes))
	resp.Body = body
	resp.ContentLength = int64(len(respBytes))
	resp.Header.Set("Content-Length", strconv.Itoa(len(respBytes)))

	if t.Cache != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 && len(respBytes) > 0 {
		if duration, err := time.ParseDuration("10s"); err == nil {
			wholeRespBytes, _ := httputil.DumpResponse(resp, true)

			go t.Cache.Set(hash, wholeRespBytes, duration)
		}
	}

	return resp, nil
}

func addCorsHeader(res http.ResponseWriter) {
	headers := res.Header()
	headers.Add("X-Cors", "Yes")
	headers.Add("Access-Control-Allow-Origin", "*")
	headers.Add("Access-Control-Allow-Headers", "Content-Type,Origin,Accept,Token,Authorization,X-App,X-Index")
	headers.Add("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
}

func NewBasicReverseProxyHandler(ctx *ReverseProxyHandlerContext) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req * http.Request) {

		// This is called BEFORE the RoundTrip intercept is via ServeHTTP

		if req.Method == "OPTIONS" {
			addCorsHeader(res)
			res.WriteHeader(http.StatusOK)
			return
		}

		req.URL.Host = ctx.Target.Host
		req.URL.Scheme = ctx.Target.Scheme
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Header.Set("X-Forwarded-Origin", req.Header.Get("Origin"))
		req.Header.Set("X-Proxy", "Zazu")
		req.Host = ctx.Target.Host

		ctx.Proxy.ServeHTTP(res, req)
	}
}