FROM golang:latest

ADD . /go/src/github.com/avesanen/dropdead

WORKDIR /go/src/github.com/avesanen/dropdead

RUN go get

RUN go build .

CMD ["./dropdead", "-c", "/etc/dropdead/dropdead.yaml"]