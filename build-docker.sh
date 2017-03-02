!/bin/bash

set -e -x

GO_VER=${GO_VER:-1.8.0}

docker run -it -v "${GOPATH}":/gopath -v "$(pwd)":/app -e "GOPATH=/gopath" -w /app golang:$GO_VER sh -c 'go build -o gorethinkdb ./'

docker build -t asia.gcr.io/kickstarter-160204/gorethinkdb .

gcloud docker -- push asia.gcr.io/kickstarter-160204/gorethinkdb

# docker run -dit --name rethinkdb-internal -p28015:28015 -p 8080:8080 rethinkdb:2.3.5;