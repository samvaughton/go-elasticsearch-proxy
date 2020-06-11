package util

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/andybalholm/brotli"
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

func DecodeResponseBodyToBytes(response *http.Response) []byte {

	// Read body to buffer
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error(fmt.Sprintf("Error reading body: %v", err))
		panic(err)
	}

	// Because go lang is a pain in the ass if you read the body then any subsequent calls
	// are unable to read the body again....
	response.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	return body
}

func DecodeResponseBodyToString(response *http.Response) string {
	return string(DecodeResponseBodyToBytes(response)[:])
}

func DecodeResponseBytes(data []byte, contentEncoding string) string {
	if len(data) == 0 {
		return ""
	}

	if contentEncoding == "" {
		return string(data)
	}

	if contentEncoding == "br" {
		// Decode brotli compression
		br := bytes.NewReader(data)

		brotliReader := brotli.NewReader(br)

		brBytes, err := ioutil.ReadAll(brotliReader)

		if err != nil {
			log.Error(err.Error())
			return ""
		}

		return string(brBytes)
	} else if contentEncoding == "gzip" {
		gzipReader, err := gzip.NewReader(bytes.NewReader(data))

		if err != nil {
			log.Error(err.Error())
			return ""
		}

		defer gzipReader.Close()

		decompressed, err := ioutil.ReadAll(gzipReader)

		if err != nil {
			log.Error(err.Error())
			return ""
		}

		return string(decompressed)
	}

	log.Error("Unsupported Content-Encoding: " + contentEncoding)

	return ""
}