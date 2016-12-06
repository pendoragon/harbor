FROM gobase:latest

MAINTAINER jiangd@vmware.com

COPY . /go/src/github.com/vmware/harbor

WORKDIR /go/src/github.com/vmware/harbor/src/jobservice

RUN go get -d github.com/docker/distribution \
    && go get -d github.com/docker/libtrust \
    && go get -d github.com/go-sql-driver/mysql \
    && go build -v -a -o /go/bin/harbor_jobservice \
    && chmod u+x /go/bin/harbor_jobservice 
WORKDIR /go/bin/
ENTRYPOINT ["/go/bin/harbor_jobservice"]
