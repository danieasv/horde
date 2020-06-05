FROM ubuntu:16.04
LABEL Author="Ståle Dahl <stalehd@telenordigital.com>"
LABEL Description="Horde UDP ingress service"
LABEL Type=Horde

# This is the RADIUS endpoint
EXPOSE 31415/udp
EXPOSE 1234/udp
EXPOSE 6000/udp
EXPOSE 7000/udp
EXPOSE 8000/udp
EXPOSE 9000/udp

# Add the Horde binary. APN and data store will be hosted in separate containers
ADD horde-udp.linux /horde-udp
ADD udp-start.sh /udp-start.sh
ENV LISTEN_PORTS=31415,1234,6000,7000,8000,9000
ENV RXTX_GRPC_ENDPOINT=localhost:15000
CMD [ "/bin/sh", "udp-start.sh" ]
