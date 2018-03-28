# Flux
Simple service for processing and saving partially parsed logs as influx metrics.

At default configuration flux expects each json-message to be in this format:
```
{
    "HOST": "some-host",
    "MESSAGE": "some log message with tag value "bla-bla-bla" to parse with value=568.7"
}
```

And can be configured to parse and save them as influx data-point:
```
some-metric,host=some-host,tag=bla-bla-bla value=568.7
```

## How to run
```bash
docker run
    -d --restart=always \
    --name flux \
    -e FLUX_INFLUX_URL=http://localhost:80 \  ## influx url
    -e FLUX_INFLUX_DB=telegraf                ## influx database
    -e FLUX_COMMIT_AMOUNT=10 \          ## commit influx bulk query after 10 records
    -e FLUX_COMMIT_INTERVAL=5 \         ## commit influx bulk query each 5 seconds
    -e FLUX_INTERNAL_BUFFER=1000 \      ## size of internal message buffer
    -e FLUX_HOST_FIELD_NAME=HOST        ## name of JSON field in log which contains hostname
    -e FLUX_MESSAGE_FIELD_NAME=MESSAGE  ## name of JSON field in log which contains log message
    -e FLUX_WORKERS=2 \                 ## amount of parallel goroutines for commiting into influx
    -v ./flux.conf:/flux.conf           ## mounting config file with metrics inside container
    -p 8080:8080 \
    ontrif/flux flux.conf               ## run flux with provided config
```

## Config file
Config file has two-level sections: routes and metrics.
Each route `route "some-name"` from config will be available as HTTP POST `/some-name` sending point.
Each route has one or more metrics with regexp to match. Each log message may be sended to one of registered routes.
Flux will check each metric from selected route in order. If one of regexp from metrics match sended log message then
flux build influx data metric point from it with help of named regexp groups.

Flux metric building logic is heavly based on regexp group's names:
* `tag_*` — any named group started with `tag_` will produce tag value for metric
* `value_*` — any named group started with `value_` will produce value for metric and will be converted to float if possible
* any other named group will be stored interanally as hidden values and can be accessed from metric's js-script

Also metric can have additional `script` property. Each script is js-script which can access and
modify `tag`, `value` and `data` global maps.

Example config file:
```
route "sync-gateway" {
    metric "sync-gateway" {
        regexp = `changes_view: Query took \b((?P<minutes>[0-9.]+)m)?((?P<seconds>[0-9.]+)s)?((?P<milliseconds>[0-9.]+)ms)?\b to return (?P<value_query_rows>\d+) rows`
        script = `
            var value = 0
            if(data.minutes) {
                value += 60 * parseFloat(data.minutes)
            }
            if(data.seconds) {
                value += parseFloat(data.seconds)
            }
            if(data.milliseconds) {
                value += parseFloat(data.milliseconds) / 1000.0
            }
            values["query_time"] = value
        `
    }
}

route "nginx" {
    metric "nginx-errors" {
        regexp = "time.*out"
        event = "timeout"
    }

    metric "nginx-access" {
        // TODO
    }
}
```

Flux will register two HTTP routes `/sync-gateway` and `/nginx`. Metric `sync-gateway` from route `sync-gateway` will match message and generates 4 named groups:

* `minutes` — this is hidden group which will not be exposed to influx automatically
* `seconds` — same
* `milliseconds` — same
* `value_query_rows` — this group will be converted to float value and exposed as `query_rows` value field

Provided script generates new value `query_time` from available data.

At the end of processing flux will generate this data point and send it to influx:
```
sync-gateway,host=some-host query_time=123.456 query_rows=12345
```

## Sending logs to flux
Each log can be sended in POST to registred route line-by-line.
Example:
```
curl -XPOST localhost:8080/nginx --data-binary @- <<EOF
{"HOST": "babycare1", "MESSAGE": "2015/11/16 21:15:21 [error] 1208#0: *4894044 upstream timed out ..."}
{"HOST": "babycare1", "MESSAGE": "2015/11/16 21:15:21 [error] 1208#0: *4894044 upstream timed out ..."}
EOF
```
