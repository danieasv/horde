FROM ubuntu:16.04
LABEL Author="Ståle Dahl <stalehd@telenordigital.com>"
LABEL Description="Horde datastore service (magpie"
LABEL Type=Horde

# This is the gRPC endpoint
EXPOSE 12002/tcp
# This is the metrics endpoint
EXPOSE 10003/tcp

# Add the Horde binary. APN and data store will be hosted in separate containers
ADD magpie.linux /magpie
ADD magpie-start.sh /magpie-start.sh

ENV DB_TYPE=sqlite3
ENV DB_CONNECTION_STRING=magpie.db?cache=shared&mode=rwc&_foreign_keys=1&_journal_mode=wal
ENV GRPC_ENDPOINT=:12002
ENV MONITORING_ENDPOINT=:10003

CMD [ "/bin/sh","/magpie-start.sh" ]