/*
 * Custom handler for apex/log, the built-in ES handler still uses index types which is now deprecated and due
 * to be removed. This handler functions in same way as the built-in one but uses the official go libraries
 */

package elasticsearch

import (
	"bytes"
	"context"
	"elasticsearch-proxy/util"
	"encoding/json"
	"fmt"
	"github.com/apex/log"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"sync"
	"time"
)

// Config for handler.
type ApexHandlerConfig struct {
	BufferSize int                  // BufferSize is the number of logs to buffer before flush (default: 100)
	IndexName  string               // Name for index
	Client     elasticsearch.Client // Client for ES
}

// defaults applies defaults to the config.
func (c *ApexHandlerConfig) defaults() {
	if c.BufferSize == 0 {
		c.BufferSize = 100
	}

	if c.IndexName == "" {
		panic("No index specified for logging")
	}
}

type Batch struct {
	Client    elasticsearch.Client
	IndexName string
	Logs      []log.Entry
}

func (b *Batch) Flush() error {
	var data bytes.Buffer

	for _, logLine := range b.Logs {
		jsonStr, err := json.Marshal(logLine)

		if err != nil {
			log.Error("Failed to marshal log entry")
		}

		data.WriteString(fmt.Sprintf(`{"index":{"_index":"%s"}}`, b.IndexName))
		data.WriteByte('\n')

		data.Write(jsonStr)
		data.WriteByte('\n')
	}

	req := esapi.BulkRequest{
		Pretty: true,
		Index:  b.IndexName,
		Body:   &data,
	}

	res, err := req.Do(context.Background(), b.Client.Transport)

	if err != nil {
		return err
	}

	if res.StatusCode >= 300 || res.StatusCode < 200 {
		fmt.Print(res)
	}

	defer res.Body.Close()

	return nil
}

func (b *Batch) Add(log *log.Entry) {
	b.Logs = append(b.Logs, *log)
}

// Handler implementation.
type Handler struct {
	*ApexHandlerConfig

	Mutex sync.Mutex
	Batch *Batch
}

// New handler with BufferSize
func NewElasticsearchHandler(config *ApexHandlerConfig) *Handler {
	config.defaults()
	return &Handler{
		ApexHandlerConfig: config,
	}
}

// HandleLog implements log.Handler.
func (h *Handler) HandleLog(e *log.Entry) error {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()

	if h.Batch == nil {
		h.Batch = &Batch{
			Client:    h.Client,
			IndexName: h.IndexName,
			Logs:      make([]log.Entry, 0),
		}
	}

	h.Batch.Add(e)

	if len(h.Batch.Logs) >= h.BufferSize {
		go h.flush(h.Batch)
		h.Batch = nil
	}

	return nil
}

// flush the given `batch` asynchronously.
func (h *Handler) flush(batch *Batch) {
	size := len(batch.Logs)
	start := time.Now()

	log.WithField("logs", size).Debug(util.LogMsg("Flushing logs"))

	if err := batch.Flush(); err != nil {
		log.WithField("logs", size).WithField("error", err.Error()).Error(util.LogMsg("Failed to flush"))
	}

	log.WithField("logs", size).WithField("time", time.Since(start)).Debug(util.LogMsg("Flush complete"))
}
