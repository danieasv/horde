FROM ubuntu:16.04
LABEL Author="Ståle Dahl <stalehd@telenordigital.com>"
LABEL Description="Horde RADIUS ingress service"
LABEL Type=Horde

# This is the RADIUS endpoint
EXPOSE 1812/udp

# Add the Horde binary. APN and data store will be hosted in separate containers
ADD horde-radius.linux /horde-radius
ADD radius-start.sh /radius-start.sh
ENV RADIUS_SHARED_SECRET=radiussharedsecret
ENV HORDE_RADIUS_ENDPOINT=localhost:11812
ENV RADIUS_ENDPOINT=:1812
CMD [ "/bin/sh", "radius-start.sh" ]