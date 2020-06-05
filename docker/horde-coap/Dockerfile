FROM ubuntu:16.04
LABEL Author="Ståle Dahl <stalehd@telenordigital.com>"
LABEL Description="Horde CoAP ingress service"
LABEL Type=Horde

# This is the RADIUS endpoint
EXPOSE 5683/udp

# Add the Horde binary. APN and data store will be hosted in separate containers
ADD horde-coap.linux /horde-coap
ADD coap-start.sh /coap-start.sh
ENV COAP_ENDPOINT=0.0.0.0:5683
ENV RXTX_GRPC_ENDPOINT=localhost:15000
CMD [ "/bin/sh", "coap-start.sh" ]
