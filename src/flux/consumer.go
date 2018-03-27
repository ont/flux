package main

import (
	"bufio"
	"encoding/json"
	"io"

	"github.com/kataras/iris/context"
	log "github.com/sirupsen/logrus"
)

type LogMessage struct {
	Host    string
	Message string
}

type Consumer struct {
	HostFieldName    string // name of "host" field in json log message
	MessageFieldName string // name of "message" field in json log message
	queue            chan *LogMessage
}

func (c *Consumer) Handle(ctx context.Context) {
	reader := bufio.NewReader(ctx.Request().Body)
	for {
		bytes, errReader := reader.ReadBytes('\n')

		if errReader != nil && errReader != io.EOF {
			log.WithError(errReader).Error("can't read line from POST body")
			return
		}

		var data map[string]interface{}

		if err := json.Unmarshal(bytes, &data); err != nil {
			log.WithError(err).WithField("line", string(bytes)).Error("can't parse line from POST body as JSON")
			return
		}

		var host, message string

		if value, ok := data[c.HostFieldName].(string); ok {
			host = value
		} else {
			log.WithField("field_name", c.HostFieldName).
				WithField("value", data[c.HostFieldName]).
				Error("can't find/convert 'host' field from JSON to string")
			return
		}

		if value, ok := data[c.MessageFieldName].(string); ok {
			message = value
		} else {
			log.WithField("field_name", c.MessageFieldName).
				WithField("value", data[c.MessageFieldName]).
				Error("can't find/convert 'message' field from JSON to string")
			return
		}

		logMessage := &LogMessage{
			Host:    host,
			Message: message,
		}

		log.WithField("message", message).Debug("consumer: sending message to queue")
		c.queue <- logMessage

		if errReader == io.EOF {
			return
		}
	}
}
