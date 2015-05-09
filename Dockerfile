FROM golang
 
ADD . $GOPATH/src/github.com/landaire/qrest

WORKDIR src/github.com/landaire/qrest

RUN go get github.com/tools/godep && /go/bin/godep go install github.com/landaire/qrest 

ENTRYPOINT ["/go/bin/qrest"]

EXPOSE 3000

