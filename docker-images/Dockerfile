FROM golang:1.13 AS builder
RUN apt-get update -y && apt-get upgrade -y && apt-get install -y ca-certificates openssh-client git curl make
RUN go get -u gopkg.in/yaml.v2
RUN go get -u github.com/LF-Engineering/ssaw/ssawsync
RUN go get -u github.com/LF-Engineering/sync-data-sources/sources
WORKDIR /go/src/github.com/LF-Engineering/sync-data-sources/sources
RUN make -f ./Makefile dockerinstall
FROM dajohn/dev-analytics-grimoire-docker-minimal
COPY docker-images/.gitconfig /root/.gitconfig
COPY docker-images/scripts/*.sh /
COPY --from=builder /go/bin/syncdatasources /usr/bin/
COPY sources/data.zip /data.zip
#COPY sources/sds.yaml /data/sds.txt
COPY docker-images/.vimrc /root/.vimrc
CMD /bin/bash
