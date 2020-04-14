package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/apex/log"
	"io/ioutil"
	"net/http"
)

func DecodeRequestBodyToBytes(request *http.Request) []byte {
	// Read body to buffer
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		log.Error(fmt.Sprintf("Error reading body: %v", err))
		panic(err)
	}

	// Because go lang is a pain in the ass if you read the body then any subsequent calls
	// are unable to read the body again....
	request.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	return body
}

func DecodeRequestBodyToString(request *http.Request) string {
	return string(DecodeRequestBodyToBytes(request)[:])
}

func NewRequestBodyJsonDecoder(request *http.Request) *json.Decoder {
	return json.NewDecoder(ioutil.NopCloser(bytes.NewBuffer(DecodeRequestBodyToBytes(request))))
}
