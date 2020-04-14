package elasticsearch

import (
	"fmt"
	"github.com/apex/log"
	"sync"
	"time"
)

// This serves to ingest all elasticsearch requests and attempt some form of debounce/de-duplication as
// simba (the front end) tends to send multiple requests that are similar and unnecessary. The "last"
// query of the debounce should be the most accurate one
// We will use the remote addr to debounce the query

type Queue struct {
	DebounceInterval time.Duration
	Channel          chan QueueLogEntry
	Items            map[string]*QueueItem
	Mutex            sync.Mutex
}

type QueueLogEntry struct {
	Key    string
	Fields log.Fields
}

type QueueItem struct {
	Addr         string
	LastReceived time.Time
	Logs         []log.Fields
}

func (qi *QueueItem) AddLog(fields log.Fields) {
	qi.Logs = append(qi.Logs, fields)
}

func NewQueue(debounce time.Duration) Queue {
	queue := Queue{
		DebounceInterval: debounce,
		Channel:          make(chan QueueLogEntry, 1000),
		Items:            make(map[string]*QueueItem),
		Mutex:            sync.Mutex{},
	}

	return queue
}

func (q *Queue) Start() {
	for {
		select {

		case queueLogEntry := <-q.Channel:

			key := queueLogEntry.Key

			if _, exists := q.Items[key]; !exists {
				q.Items[key] = &QueueItem{
					Addr:         key,
					LastReceived: time.Now(),
					Logs:         make([]log.Fields, 0),
				}
			}

			q.Items[key].AddLog(queueLogEntry.Fields)
		case <-time.After(q.DebounceInterval):
			// Now we need "flush" the logs per IP and pull the last request out of each of them
			for _, qi := range q.Items {

				if len(qi.Logs) == 0 {
					continue
				}

				// Pluck last one off the array
				lastEntry := qi.Logs[len(qi.Logs)-1]

				fields := lastEntry.Fields()

				log.WithFields(fields).Info(fmt.Sprintf(LogMsg("Added to buffer (debounced %d queries)"), len(qi.Logs)))
				EsLogger.WithFields(fields).Info(fmt.Sprintf("%v", fields.Get("url")))
			}

			// Now we reset the map as we have done our "logging"
			q.Items = make(map[string]*QueueItem)
		}
	}
}
