#
# NOTE: this Dockerfile is only for packaging, actual build steps are in .drone.yml
#
FROM alpine

# fix "/bin/sh: /stick: not found" SEE: https://stackoverflow.com/a/35613430/434255
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

ENV FLUX_INFLUX_DB telegraf
ENV FLUX_INFLUX_URL http://localhost:8086
ADD ./src/flux/flux /flux

# install root CAs
# RUN apk add --no-cache ca-certificates

EXPOSE 8080
ENTRYPOINT /flux
