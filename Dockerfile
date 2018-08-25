FROM golang:alpin as go_builder
RUN apk add --no-cache curl
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN go get github.com/zeidlermicha/go-d21s


