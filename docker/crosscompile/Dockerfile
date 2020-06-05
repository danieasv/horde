FROM ubuntu:16.04

RUN apt-get -y update && \
    apt-get install -y wget && \
    apt-get install -y gcc
RUN wget -q https://dl.google.com/go/go1.14.2.linux-amd64.tar.gz && \
    tar -xvf go1.14.2.linux-amd64.tar.gz && \
    mv go /usr/local/go
ENV GOROOT=/usr/local/go
ENV PATH=${PATH}/bin:${GOROOT}/bin:${PATH}
