FROM alpine:latest
MAINTAINER Arjun Naik <arjun.rn@gmail.com>

# add binary
ADD build/linux/simple-scaler /

ENTRYPOINT ["/simple-scaler"]
