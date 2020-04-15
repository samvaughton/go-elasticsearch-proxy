package elasticsearch

import "github.com/apex/log"

/*
 * Filters determine whether or not this query should be logged, some queries may be filtered out because they have no metrics
 * whereas some might match a host or an IP and be filtered out
 */

type FilterProcessor struct {
	Filters []func(fields log.Fields) bool
}

func NewFilterProcessor() FilterProcessor {
	return FilterProcessor{
		Filters: make([]func(fields log.Fields) bool, 0),
	}
}

func (fp *FilterProcessor) AddFilter(filter func(fields log.Fields) bool) {
	fp.Filters = append(fp.Filters, filter)
}

func (fp *FilterProcessor) Process(fields log.Fields) bool {
	// If a single filter returns false, then we stop execution and don't process
	for _, filter := range fp.Filters {
		if filter(fields) == false {
			return false
		}
	}

	return true
}
