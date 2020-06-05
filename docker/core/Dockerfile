FROM ubuntu:16.04
LABEL Author="Ståle Dahl <stalehd@telenordigital.com>"
LABEL Description="Horde core service"
LABEL Type=Horde

# This is the API port
EXPOSE 8080/tcp
# This is the metrics endpoint
EXPOSE 10001/tcp
# This is the gRPC rxtx server
EXPOSE 15000/tcp
# This is the gRPC management port server
EXPOSE 1234/tcp

# This is optional. Run this if you plan to do requests to hosts with
# updated SSL certificates. Ubuntu16.04 is decidedly ancient so we need an
# updated list of certificates. This is relatively slow when building the
# image so it can be skipped if you only do the release tests.
RUN apt-get update
RUN apt-get install -y ca-certificates

# Add the Horde binary. APN and data store will be hosted in separate containers
ADD horde.linux /horde
ADD horde-start.sh /horde-start.sh


ENV GITHUB_CLIENT_ID=000000000
ENV GITHUB_CLIENT_SECRET=00000000
ENV CONNECT_HOST=connect.staging.telenordigital.com
ENV CONNECT_CLIENT_ID=telenordigital-connectexample-web
ENV CONNECT_CLIENT_SECRET=
ENV WORKER_ID=1
ENV DATACENTER_ID=1
ENV DATASTORE_ENDPOINT=localhost:6666

# These match the exposed ports mentioned above
ENV APN_SERVER_ENDPOINT=:12000
ENV MONITORING_ENDPOINT=:10001
ENV HTTP_ENDPOINT=:8080

ENV DB_TYPE=sqlite3
ENV DB_CONNECTION_STRING=horde.db?cache=shared&mode=rwc&_foreign_keys=1&_journal_mode=wal
ENV MANAGEMENT_ENDPOINT=0.0.0.0:1234

ENV RADIUS_GRPC_ENDPOINT=0.0.0.0:11812

CMD [ "/bin/sh","/horde-start.sh" ]