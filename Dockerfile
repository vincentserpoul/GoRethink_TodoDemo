FROM busybox:ubuntu-14.04

MAINTAINER Vincent Serpoul "<vincent@serpoul.com>"

# admin, http, udp, cluster, graphite, opentsdb, collectd
EXPOSE 3000

WORKDIR /app

# copy binary into image
COPY gorethinkdb /app/

COPY templates /app/templates
COPY static /app/static

# Add influxd to the PATH
ENV PATH=/app:$PATH

ENTRYPOINT ["gorethinkdb"]