package main

import (
	"github.com/kataras/iris"
	log "github.com/sirupsen/logrus"
)

type RootConsumer struct {
	RouteFieldName string
	consumers      map[string]*Consumer // route name to consumer map

	BaseConsumer
}

func (c *RootConsumer) Handle(ctx iris.Context) {
	for _, message := range c.parseJSONs(ctx.Request().Body) {
		var route string

		if value, ok := message.data[c.RouteFieldName].(string); ok {
			route = value
		} else {
			log.WithField("field_name", c.RouteFieldName).
				WithField("value", message.data[c.RouteFieldName]).
				Error("can't find/convert 'route' field from JSON to string")
			continue
		}

		if consumer, found := c.consumers[route]; found {
			log.WithField("message", message).WithField("route", route).Debug("consumer: sending message to queue of route")
			consumer.queue <- message
		} else {
			log.WithField("message", message).WithField("route", route).Error("consumer: can't found consumer for route")
		}
	}
}

func (c *RootConsumer) AddConsumer(route string, consumer *Consumer) {
	c.consumers[route] = consumer
}
