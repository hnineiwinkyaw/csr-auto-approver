FROM golang:1.16.6
RUN apt-get update
RUN apt-get install -y vim curl ca-certificates

COPY . /go/src/code/
WORKDIR /go/src/code/

RUN export GOPATH=/go/src

RUN go mod download

RUN ls /go

RUN ls /go/src

RUN ls /go/pkg

RUN go build 

CMD ["go","run","main.go"]
