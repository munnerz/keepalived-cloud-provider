FROM alpine:3.5

ADD keepalived-cloud-provider /usr/local/bin/keepalived-cloud-provider

CMD ["keepalived-cloud-provider"]
