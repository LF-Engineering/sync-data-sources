FROM golang:1.15 AS builder
RUN apt-get update -y && apt-get upgrade -y && apt-get install -y ca-certificates openssh-client git curl make
RUN go get -u gopkg.in/yaml.v2
RUN go get -u github.com/json-iterator/go
RUN git clone --branch v7.10.0 https://github.com/elastic/go-elasticsearch.git $GOPATH/src/github.com/elastic/go-elasticsearch/v7
RUN go get -u github.com/LF-Engineering/ssaw/ssawsync
RUN go get -u github.com/LF-Engineering/sync-data-sources/sources
RUN go get -u github.com/PuerkitoBio/goquery
RUN go get -u github.com/avast/retry-go
RUN go get -u github.com/LF-Engineering/da-ds
RUN CGO_ENABLED=0 go install -ldflags '-s' github.com/LF-Engineering/da-ds/cmd/dads
WORKDIR /go/src/github.com/LF-Engineering/sync-data-sources/sources
RUN make -f ./Makefile dockerinstall
# The final version should not use bthis base image, so it needs to have 'cloc' installed.
FROM dajohn/dev-analytics-grimoire-docker-minimal
COPY docker-images/.gitconfig /root/.gitconfig
COPY docker-images/scripts/*.sh /
COPY --from=builder /go/bin/syncdatasources /usr/bin/
COPY --from=builder /go/bin/dads /usr/bin/
COPY sources/data.zip /data.zip
COPY sources/uuid.py /usr/bin/
COPY sources/gitops.py /usr/bin/
#COPY sources/sds.yaml /data/sds.txt
COPY docker-images/.vimrc /root/.vimrc
CMD /bin/bash
