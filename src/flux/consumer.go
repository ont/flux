package main

import (
	"github.com/kataras/iris/context"
	log "github.com/sirupsen/logrus"
)

type Consumer struct {
	queue chan *LogMessage

	BaseConsumer
}

func (c *Consumer) Handle(ctx context.Context) {
	for _, message := range c.parseJSONs(ctx.Request().Body) {
		log.WithField("message", message).Debug("consumer: sending message to queue")
		c.queue <- message
	}
}
