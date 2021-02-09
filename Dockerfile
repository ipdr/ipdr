FROM alpine:3.12 as build

RUN apk add -U curl wget ca-certificates

RUN wget $(curl -s https://api.github.com/repos/ipdr/ipdr/releases/latest | grep "browser_download_url.*linux.*amd64.tar.gz" | cut -d : -f 2,3 | tr -d \")
RUN tar zxvf *.tar.gz
RUN mv ipdr /ipdr

FROM alpine:3.12
COPY --from=build /ipdr /usr/bin/ipdr
ENTRYPOINT ["ipdr"]
