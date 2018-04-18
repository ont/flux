package main

import (
	"os"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/kataras/iris"
	"github.com/mohae/deepcopy"
	log "github.com/sirupsen/logrus"
)

func NewRootConsumer(app *iris.Application) *RootConsumer {
	consumer := &RootConsumer{
		BaseConsumer: BaseConsumer{
			HostFieldName:    GetenvStr("FLUX_HOST_FIELD_NAME", "HOST"),       // default value as in syslog message
			MessageFieldName: GetenvStr("FLUX_MESSAGE_FIELD_NAME", "MESSAGE"), // ...
		},
		RouteFieldName: GetenvStr("FLUX_ROUTE_FIELD_NAME", "ROUTE"),
		consumers:      make(map[string]*Consumer),
	}

	app.Post("/", consumer.Handle)

	return consumer
}

func NewConsumer(app *iris.Application, route *Route) (*Consumer, chan *LogMessage) {
	queue := make(chan *LogMessage, GetenvInt("FLUX_INTERNAL_BUFFER", 1000))

	consumer := &Consumer{
		BaseConsumer: BaseConsumer{
			HostFieldName:    GetenvStr("FLUX_HOST_FIELD_NAME", "HOST"),       // default value as in syslog message
			MessageFieldName: GetenvStr("FLUX_MESSAGE_FIELD_NAME", "MESSAGE"), // ...
		},
		queue: queue,
	}

	app.Post(route.Name, consumer.Handle)

	return consumer, queue
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

	// Make clone of all metrics and compile their scripts.
	// So every worker recieves non-shared goja.Runtime and goja.Program in metric.script
	// NOTE: this part fixes race-condition crashes during concurrent RunProgram on single goja.Runtime
	cmetrics := make([]*Metric, 0)
	for _, metric := range metrics {
		cmetric, ok := deepcopy.Copy(metric).(*Metric)
		if !ok {
			log.Fatal("can't do deepclone for metric")
		}
		cmetric.prepareScript()
		cmetrics = append(cmetrics, cmetric)
	}

	worker := &Worker{
		CommitInterval: time.Duration(commitInterval) * time.Second,
		CommitAmount:   GetenvInt("FLUX_COMMIT_AMOUNT", 10),
		Database:       os.Getenv("FLUX_INFLUX_DB"),

		queue:   queue,
		client:  client,
		metrics: cmetrics,
	}

	worker.CreateBatch()
	return worker
}
