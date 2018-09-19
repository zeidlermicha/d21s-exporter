FROM golang:alpine as go_builder
RUN apk update && apk upgrade && \
    apk add --no-cache curl git
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
WORKDIR /go/src/github.com/zeidlermicha
RUN git clone https://github.com/zeidlermicha/go-d21s.git
RUN git clone https://github.com/zeidlermicha/d21s-exporter.git
WORKDIR /go/src/github.com/zeidlermicha/d21s-exporter
RUN dep ensure
RUN go build -o d21s-exporter d21s-exporter.go

FROM alpine:latest
WORKDIR /usr/local/bin
RUN apk update && apk upgrade && \
    apk add --no-cache ca-certificates
COPY --from=go_builder /go/src/github.com/zeidlermicha/d21s-exporter/d21s-exporter .
ENTRYPOINT ["/usr/local/bin/d21s-exporter"]