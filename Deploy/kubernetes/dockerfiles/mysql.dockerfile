FROM mysql:5.6

WORKDIR /tmp

ADD ./Deploy/db/registry.sql r.sql 

ADD ./Deploy/db/docker-entrypoint.sh /entrypoint.sh
RUN chmod u+x /entrypoint.sh 