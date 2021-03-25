FROM golang:1.15 AS builder
ARG BRANCH
RUN apt-get update -y && apt-get upgrade -y && apt-get install -y ca-certificates openssh-client git curl make
RUN go get -u gopkg.in/yaml.v2
RUN go get -u golang.org/x/oauth2
RUN go get -u golang.org/x/text/transform
RUN go get -u golang.org/x/text/unicode/norm
RUN go get -u github.com/json-iterator/go
RUN go get -u github.com/LF-Engineering/dev-analytics-libraries/auth0
RUN go get -u github.com/LF-Engineering/dev-analytics-libraries/elastic
RUN go get -u github.com/LF-Engineering/dev-analytics-libraries/http
RUN go get -u github.com/LF-Engineering/dev-analytics-libraries/slack
RUN go get -u github.com/google/go-github/github
RUN go get -u github.com/PuerkitoBio/goquery
RUN go get -u github.com/avast/retry-go
RUN go get -u github.com/dgrijalva/jwt-go
RUN go get -u github.com/google/uuid
WORKDIR /go/src/github.com/LF-Engineering
RUN git clone https://github.com/LF-Engineering/da-ds
RUN git clone https://github.com/LF-Engineering/sync-data-sources
WORKDIR /go/src/github.com/LF-Engineering/da-ds
RUN CGO_ENABLED=0 go install -ldflags '-s' github.com/LF-Engineering/da-ds/cmd/dads
WORKDIR /go/src/github.com/LF-Engineering/sync-data-sources/sources
RUN echo "Building for branch ${BRANCH}"
RUN git checkout ${BRANCH}
RUN git pull
RUN make -f ./Makefile dockerinstall
# The final version should not use this base image, so it needs to have 'cloc' installed.
FROM dajohn/dev-analytics-grimoire-docker-minimal
COPY docker-images/.gitconfig /root/.gitconfig
COPY docker-images/scripts/*.sh /
COPY --from=builder /go/bin/syncdatasources /usr/bin/
COPY --from=builder /go/bin/dads /usr/bin/
COPY sources/data.zip /data.zip
COPY sources/uuid.py /usr/bin/
COPY sources/gitops.py /usr/bin/
COPY sources/detect-removed-commits.sh /usr/bin/
#COPY sources/sds.yaml /data/sds.txt
COPY docker-images/.vimrc /root/.vimrc
CMD /bin/bash
