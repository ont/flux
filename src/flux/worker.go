package main

import (
	"strconv"
	"strings"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"
	log "github.com/sirupsen/logrus"
)

type Worker struct {
	CommitInterval time.Duration
	CommitAmount   int
	Database       string

	queue   chan *LogMessage
	batch   influx.BatchPoints
	client  influx.Client
	metrics []*Metric
	script  *Script
}

func (w *Worker) Start() {
	tick := time.Tick(w.CommitInterval)

	for {
		select {
		case line, ok := <-w.queue:
			if !ok {
				w.Flush()
				return // queue closed, we can exit
			}

			w.Process(line)

		case <-tick:
			w.Flush() // send bulk query to influx every tick event
		}
	}
}

func (w *Worker) CreateBatch() {
	batch, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  w.Database,
		Precision: "s",
	})

	if err != nil {
		log.WithError(err).Fatal("can't create new influx batch")
	}

	w.batch = batch
}

func (w *Worker) Process(message *LogMessage) {
	for _, metric := range w.metrics {
		matches := metric.re.FindStringSubmatch(message.Message)
		if len(matches) > 0 {

			log.WithField("matches", matches).Debug("worker: found matches")

			tags, values, data, err := w.GetTagsValues(message.Message, metric, matches)

			log.WithField("tags", tags).
				WithField("values", values).
				WithField("data", data).Debug("worker: parsed tags, values and data")

			if err != nil {
				break
			}

			if metric.script != nil {
				tags, values, err = w.ProcessScript(metric.script, tags, values, data)
				if err != nil {
					break
				}

				log.WithField("tags", tags).
					WithField("values", values).
					Debug("worker: parsed tags and values after script")
			}

			// add hostname as tag to point
			// NOTE: it overwrites any "tag_host" value from regexp and script
			tags["host"] = message.Host

			log.WithField("tags", tags).
				WithField("values", values).
				Debug("worker: final tags and values for point")

			w.AddPoint(metric, tags, values)

			break // ignore any other metrics from config
		}
	}

	if len(w.batch.Points()) >= w.CommitAmount {
		w.Flush()
	}
}

// GetTagsValues extracts tags, values and additional data from regexp match result
func (w *Worker) GetTagsValues(logMessage string, metric *Metric, matches []string) (tags PointTags, values PointValues, data PointValues, err error) {
	tags = make(PointTags)
	values = make(PointValues)
	data = make(PointValues)

	if metric.eventName != "" {
		values[metric.eventName] = 1 // just one value-column with "1" value
	}

	for i, name := range metric.re.SubexpNames() {
		switch {
		// save every "tag_asdfasdf" regexp group as "asdfasdf" tag-field
		case strings.HasPrefix(name, "tag_"):
			tags[name[4:]] = matches[i]

		// save every "value_asdfasdf" regexp group as "asdfasdf" value-field
		case strings.HasPrefix(name, "value_"):
			value, err := strconv.ParseFloat(matches[i], 64)

			if err != nil {
				log.WithField("value", matches[i]).
					WithField("name", name).
					WithField("message", logMessage).
					WithError(err).
					Error("can't parse value as float")
				return nil, nil, nil, err
			}

			values[name[6:]] = value

		default:
			data[name] = matches[i]
		}
	}

	return tags, values, data, nil
}

func (w *Worker) ProcessScript(script *Script, tags PointTags, values, data PointValues) (PointTags, PointValues, error) {
	script.Tags = tags
	script.Values = values
	script.Data = data
	err := script.Process()
	return script.Tags, script.Values, err
}

func (w *Worker) AddPoint(metric *Metric, tags map[string]string, values map[string]interface{}) {
	point, err := influx.NewPoint(metric.Name, tags, values, time.Now())
	if err != nil {
		log.WithError(err).Fatal("can't create influx point")
	}
	w.batch.AddPoint(point)
}

func (w *Worker) Flush() {
	log.WithField("count", len(w.batch.Points())).Debug("flushing to influx")

	if err := w.client.Write(w.batch); err != nil {
		log.WithError(err).Error("can't write batch to influx, dropping batch...")
	}

	w.CreateBatch()
}
