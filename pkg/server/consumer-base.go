package server

import (
	"bufio"
	"encoding/json"
	"io"

	log "github.com/sirupsen/logrus"
)

type BaseConsumer struct{}

func (c *BaseConsumer) parseJSONs(body io.ReadCloser) []LogMessage {
	messages := make([]LogMessage, 0) // TODO: change to channel (don't parse all json messages into memory)

	reader := bufio.NewReader(body)
	for {
		bytes, errReader := reader.ReadBytes('\n')

		if errReader != nil && errReader != io.EOF {
			log.WithError(errReader).Error("can't read line from POST body")
			break
		}

		var message LogMessage

		if err := json.Unmarshal(bytes, &message); err != nil {
			log.WithError(err).WithField("line", string(bytes)).Error("can't parse line from POST body as JSON")
			goto eofcheck
		}

		if !message.Validate() {
			log.WithField("line", string(bytes)).Error("skip invalid message")
			goto eofcheck
		}

		messages = append(messages, message)

	eofcheck:
		if errReader == io.EOF {
			break
		}
	}

	return messages
}
