package main

import (
	"os"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/kataras/iris"
	log "github.com/sirupsen/logrus"
)

func NewConsumer(app *iris.Application, route *Route) chan *LogMessage {
	queue := make(chan *LogMessage, GetenvInt("FLUX_INTERNAL_BUFFER", 1000))

	consumer := &Consumer{
		HostFieldName:    GetenvStr("FLUX_HOST_FIELD_NAME", "HOST"),       // default value as in syslog message
		MessageFieldName: GetenvStr("FLUX_MESSAGE_FIELD_NAME", "MESSAGE"), // ...
		queue:            queue,
	}

	app.Post(route.Name, consumer.Handle)

	return queue
}

func NewWorkers(queue chan *LogMessage, metrics []*Metric) []*Worker {
	cnt := GetenvInt("FlUX_WORKERS", 2)
	workers := make([]*Worker, 0, cnt)

	for i := 0; i < cnt; i++ {
		workers = append(workers, NewWorker(queue, metrics))
	}

	return workers
}

func NewWorker(queue chan *LogMessage, metrics []*Metric) *Worker {
	client, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr: os.Getenv("FLUX_INFLUX_URL"), //"http://localhost:8086"
	})

	if err != nil {
		log.WithError(err).Fatal("can't connect to influx")
	}

	commitInterval := GetenvInt("FLUX_COMMIT_INTERVAL", 5)

	worker := &Worker{
		CommitInterval: time.Duration(commitInterval) * time.Second,
		CommitAmount:   GetenvInt("FLUX_COMMIT_AMOUNT", 10),
		Database:       os.Getenv("FLUX_INFLUX_DB"),

		queue:   queue,
		client:  client,
		metrics: metrics,
	}

	worker.CreateBatch()
	return worker
}
