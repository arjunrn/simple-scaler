FROM alpine:latest
MAINTAINER Arjun Naik <arjun.rn@gmail.com>

# add binary
ADD build/linux/dumb-scaler /

ENTRYPOINT ["/dumb-scaler"]
