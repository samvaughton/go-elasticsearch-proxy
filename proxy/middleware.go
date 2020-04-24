package proxy

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strconv"
)

type MiddlewareTransport struct {
	http.RoundTripper
	ReverseProxyHandlerContext *ReverseProxyHandlerContext
	MiddlewareRoutine func(ctx ReverseProxyHandlerContext, req *http.Request, resp *http.Response, decodedRequestBody string, decodedResponseBody string)
}

func (t *MiddlewareTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	var decodedRequestBody []byte

	if req.URL != nil && req.Header != nil  && req.Body != nil {
		decodedRequestBody = DecodeRequestBodyToBytes(req)
	}

	resp, err = t.RoundTripper.RoundTrip(req)

	if err != nil {
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
			DecodeResponseBytes(
				respBytes,
				resp.Header.Get("Content-Encoding"),
			),
		)
	}

	body := ioutil.NopCloser(bytes.NewReader(respBytes))
	resp.Body = body
	resp.ContentLength = int64(len(respBytes))

	resp.Header.Set("Content-Length", strconv.Itoa(len(respBytes)))
	resp.Header.Set("X-Proxy", "Zazu")

	return resp, nil
}


func NewBasicReverseProxyHandler(ctx *ReverseProxyHandlerContext) func(res http.ResponseWriter, req * http.Request) {
	return func(res http.ResponseWriter, req * http.Request) {
		req.URL.Host = ctx.Target.Host
		req.URL.Scheme = ctx.Target.Scheme
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Header.Set("X-Forwarded-Origin", req.Header.Get("Origin"))
		req.Header.Set("X-Proxy", "Zazu")
		req.Host = ctx.Target.Host

		ctx.Proxy.ServeHTTP(res, req)
	}
}