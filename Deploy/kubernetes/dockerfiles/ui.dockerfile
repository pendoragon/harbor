FROM debian:jessie

RUN apt-get update \
    && apt-get install -y libldap2-dev \
    && rm -r /var/lib/apt/lists/*

COPY Deploy/kubernetes/dockerfiles/bin/harbor_ui /go/bin/harbor_ui
COPY views /go/bin/views
COPY static /go/bin/static
COPY favicon.ico /go/bin/favicon.ico
COPY Deploy/jsminify.sh /tmp/jsminify.sh

RUN chmod u+x /go/bin/harbor_ui \
    && sed -i 's/TLS_CACERT/#TLS_CAERT/g' /etc/ldap/ldap.conf \
    && sed -i '$a\TLS_REQCERT allow' /etc/ldap/ldap.conf \
    && /tmp/jsminify.sh /go/bin/views/sections/script-include.htm /go/bin/static/resources/js/harbor.app.min.js

WORKDIR /go/bin/
ENTRYPOINT ["/go/bin/harbor_ui"]

EXPOSE 80