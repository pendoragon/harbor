FROM gobase:latest

MAINTAINER jiangd@vmware.com

COPY src/. /go/src/github.com/vmware/harbor/src
WORKDIR /go/src/github.com/vmware/harbor/src/ui

RUN go build -v -a -o /go/bin/harbor_ui

ENV MYSQL_USR root \
    MYSQL_PWD root \
    REGISTRY_URL localhost:5000

COPY src/ui/views /go/bin/views
COPY src/ui/static /go/bin/static
COPY src/favicon.ico /go/bin/favicon.ico
COPY make/jsminify.sh /tmp/jsminify.sh

RUN chmod u+x /go/bin/harbor_ui \
    && sed -i 's/TLS_CACERT/#TLS_CAERT/g' /etc/ldap/ldap.conf \
    && sed -i '$a\TLS_REQCERT allow' /etc/ldap/ldap.conf \
    && /tmp/jsminify.sh /go/bin/views/sections/script-include.htm /go/bin/static/resources/js/harbor.app.min.js /go/bin/

WORKDIR /go/bin/
ENTRYPOINT ["/go/bin/harbor_ui"]

EXPOSE 80

