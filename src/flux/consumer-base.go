package main

import (
	"bufio"
	"encoding/json"
	"io"

	log "github.com/sirupsen/logrus"
)

type BaseConsumer struct {
	HostFieldName    string // name of "host" field in json log message
	MessageFieldName string // name of "message" field in json log message
}

type LogMessage struct {
	Host    string
	Message string

	data map[string]interface{}
}

func (c *BaseConsumer) parseJSONs(body io.ReadCloser) []*LogMessage {
	messages := make([]*LogMessage, 0) // TODO: change to channel (don't parse all json messages into memory)

	reader := bufio.NewReader(body)
	for {
		bytes, errReader := reader.ReadBytes('\n')

		if errReader != nil && errReader != io.EOF {
			log.WithError(errReader).Error("can't read line from POST body")
			break
		}

		var data map[string]interface{}
		var host, message string
		var logMessage *LogMessage

		if err := json.Unmarshal(bytes, &data); err != nil {
			log.WithError(err).WithField("line", string(bytes)).Error("can't parse line from POST body as JSON")
			goto eofcheck
		}

		if value, ok := data[c.HostFieldName].(string); ok {
			host = value
		} else {
			log.WithField("field_name", c.HostFieldName).
				WithField("value", data[c.HostFieldName]).
				Error("can't find/convert 'host' field from JSON to string")
			goto eofcheck
		}

		if value, ok := data[c.MessageFieldName].(string); ok {
			message = value
		} else {
			log.WithField("field_name", c.MessageFieldName).
				WithField("value", data[c.MessageFieldName]).
				Error("can't find/convert 'message' field from JSON to string")
			goto eofcheck
		}

		logMessage = &LogMessage{
			Host:    host,
			Message: message,
			data:    data,
		}

		messages = append(messages, logMessage)

	eofcheck:
		if errReader == io.EOF {
			break
		}
	}

	return messages
}
