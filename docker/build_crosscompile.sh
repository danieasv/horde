# This builds a docker image that can crosscompile linux binaries. The
# official docker images uses a fairly new version of libc that isn't
# compatible with Ubuntu LTS and AWS AMI.
docker build crosscompile --tag ee/cross:go1.14.2
