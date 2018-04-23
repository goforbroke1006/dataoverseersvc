FROM debian:jessie

RUN apt-get update
RUN apt-get install -y git curl wget tar zip gzip

ARG GO_DIST=go1.9.1.linux-amd64.tar.gz
RUN cd /
RUN wget https://storage.googleapis.com/golang/${GO_DIST}
RUN tar -C /usr/local -xvf ${GO_DIST}
RUN rm -f ${GO_DIST}

ENV GOROOT  "/usr/local/go"
ENV PATH    "$PATH:/usr/local/go/bin"

RUN mkdir -p /go/bin /go/pkg /go/src
RUN chmod -R 0777 /go/

ENV GOPATH  "/go"
ENV PATH    "$PATH:/go/bin"

RUN go get -u github.com/golang/dep/cmd/dep

WORKDIR /code