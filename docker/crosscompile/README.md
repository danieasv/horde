# Cross compiling docker image

This is for ubuntu 16.04 and go 1.13.4 (at time of writing) since most other images
use a newer version of libc.

This image is used when running `make linux`. Native linux builds don't need this.