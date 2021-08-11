FROM golang:1.16.6 as builder
COPY main.go /go/src
WORKDIR /go/src
RUN go mod init app
RUN go mod tidy


FROM golang:1.16.6 
RUN apt-get update
RUN apt-get install -y vim curl ca-certificates
COPY . /go/src/code/
WORKDIR /go/src/code/
ENV GO111MODULE=on
COPY --from=builder /go/src/go.mod /go/src/code/
COPY --from=builder /go/src/go.sum /go/src/code/
COPY --from=builder /go/pkg/mod /go/src
CMD ["go","run","main.go"]