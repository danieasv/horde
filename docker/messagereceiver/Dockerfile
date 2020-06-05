FROM ubuntu:16.04
LABEL Author="Ståle Dahl <stalehd@telenordigital.com>"
LABEL Description="Messagereceiver"
LABEL Type=Horde

EXPOSE 4711/udp
EXPOSE 4712/udp
EXPOSE 8282/tcp

# Add the Horde binary. APN and data store will be hosted in separate containers
ADD messagereceiver.linux /messagereceiver
ADD messagereceiver-start.sh /messagereceiver-start.sh
CMD [ "/bin/sh", "messagereceiver-start.sh" ]