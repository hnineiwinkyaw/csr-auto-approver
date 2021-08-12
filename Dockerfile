FROM golang:1.16.6 
RUN apt-get update
RUN apt-get install -y vim curl ca-certificates
COPY main.go /go/src/code/
WORKDIR /go/src/code/
RUN go mod init app
RUN go mod tidy
ENV GO111MODULE=on
CMD ["go","run","main.go"]