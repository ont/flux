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
Each log can be send in POST to registred route line-by-line.
Example:
```
curl -XPOST localhost:8080/nginx --data-binary @- <<EOF
{"HOST": "babycare1", "MESSAGE": "2015/11/16 21:15:21 [error] 1208#0: *4894044 upstream timed out ..."}
{"HOST": "babycare1", "MESSAGE": "2015/11/16 21:15:21 [error] 1208#0: *4894044 upstream timed out ..."}
EOF
```

## Special "/" route
Flux also has special root route `/` which can consume all messages. It is especially usefull when using flux with syslog-ng.

Message for this route must have additional "route" field (`ROUTE` by default, but can be configured with `FLUX_ROUTE_FIELD_NAME` env var).

Example of message
```
{
    "HOST": "some-host",
    "MESSAGE": "some log message with tag value "bla-bla-bla" to parse with value=568.7",
    "ROUTE": "sync-gateway"
}
```

Send it with
```
curl -XPOST localhost:8080/ --data-binary @- <<EOF
{ "HOST": "some-host", "MESSAGE": "some log message with tag value "bla-bla-bla" to parse with value=568.7", "ROUTE": "sync-gateway" }
EOF
```

This message will be processed with metrics from `sync-gateway` route from config.

## How to use with syslog-ng
Example config for syslog-ng which listen for rfc5424 (non-json) log messages on port 5555 and sends sync-gateway related messages to flux:

```
@version: 3.14
@include "scl.conf"

options { 
    chain-hostnames(off); 
    #use-dns(no); 
    #use-fqdn(no);
    log-msg-size(32768);
};

source s_net {
    ## rfc5424 with frames which works with "logger" system utility
    network(transport("tcp") port("5555") flags(syslog-protocol));
};

destination d_flux {
    http(
        url("localhost:8080/")
        method("POST")
        body("$(format-json --key HOST,MESSAGE,ROUTE)")
    );
};

destination d_backup { file("/mnt/logs/all-${YEAR}-${MONTH}-${DAY}.log"); };

filter f_flux { tags("flux"); };

## sync-gateway related filter + rewrite
filter f_sg { program('sync-gateway'); };
rewrite r_sg {
    set-tag("flux");
    set("sync-gateway", value("ROUTE"));
};


log {
    ## 0. recieve logs from rfc5424 source
    source(s_net);

    ## 1. normalize: extract fields, parse json, transfrom message...
    junction {
        channel { filter(f_sg); rewrite(r_sg); flags(final); };
        # channel { parser(p_json); rewrite(r_json); flags(final); };
        # channel { rewrite(r_plain); flags(final); };
    };

    ## 2. backup all logs to disk
    destination(d_backup);

    ## 3. log any message to elasticsearch (SEE: github.com/ont/stick)
    # destination(d_stick);

    ## 4. send metrics to influx
    log {
        filter(f_flux);
        destination(d_flux);
    };
};
```
