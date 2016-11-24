FROM debian:jessie

RUN apt-get update \
    && apt-get install -y libldap2-dev \
    && rm -r /var/lib/apt/lists/*

COPY Deploy/kubernetes/dockerfiles/bin/harbor_jobservice /go/bin/harbor_jobservice

RUN chmod u+x /go/bin/harbor_jobservice 

WORKDIR /go/bin/
ENTRYPOINT ["/go/bin/harbor_jobservice"]