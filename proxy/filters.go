package proxy

import (
	"github.com/apex/log"
	"net/http"
)

/*
 * Filters determine whether or not this query should be logged, some queries may be filtered out because they have no metrics
 * whereas some might match a host or an IP and be filtered out
 */

type FilterProcessor struct {
	Filters []func(req *http.Request, fields log.Fields) bool
}

func NewFilterProcessor() FilterProcessor {
	return FilterProcessor{
		Filters: make([]func(req *http.Request, fields log.Fields) bool, 0),
	}
}

func (fp *FilterProcessor) AddFilter(filter func(req *http.Request, fields log.Fields) bool) {
	fp.Filters = append(fp.Filters, filter)
}

func (fp *FilterProcessor) Process(req *http.Request, fields log.Fields) bool {
	// If a single filter returns false, then we stop execution and don't process
	for _, filter := range fp.Filters {
		if filter(req, fields) == false {
			return false
		}
	}

	return true
}
