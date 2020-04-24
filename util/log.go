package util

import (
	"fmt"
	"github.com/apex/log"
	"time"
)

func LogData(fields *log.Fields) {
	log.WithFields(fields).Debug("Generic Request")
}

func LogMsg(message string) string {
	return fmt.Sprintf("%s Logger: %s", time.Now(), message)
}
