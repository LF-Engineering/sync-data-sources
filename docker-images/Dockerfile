FROM golang:1.15 AS builder
ARG BRANCH
RUN apt-get update -y && apt-get upgrade -y && apt-get install -y ca-certificates openssh-client git curl make
WORKDIR /go/src/github.com/LF-Engineering
RUN git clone https://github.com/LF-Engineering/gitops
WORKDIR /go/src/github.com/LF-Engineering/gitops
RUN CGO_ENABLED=0 go install -ldflags "-s -w -extldflags '-static'" github.com/LF-Engineering/gitops
WORKDIR /go/src/github.com/LF-Engineering
RUN git clone https://github.com/LF-Engineering/gitops
WORKDIR /go/src/github.com/LF-Engineering/gitops
RUN CGO_ENABLED=0 go install -ldflags "-s -w -extldflags '-static'" github.com/LF-Engineering/gitops
WORKDIR /go/src/github.com/LF-Engineering
RUN git clone https://github.com/LF-Engineering/da-ds
RUN git clone https://github.com/LF-Engineering/sync-data-sources
WORKDIR /go/src/github.com/LF-Engineering/da-ds
RUN CGO_ENABLED=0 go install -ldflags "-s -w -extldflags '-static' -X build.GitCommit=$(git rev-parse --verify HEAD)" github.com/LF-Engineering/da-ds/cmd/dads
WORKDIR /go/src/github.com/LF-Engineering/sync-data-sources/sources
RUN echo "Building for branch ${BRANCH}"
RUN git checkout -b ${BRANCH} "origin/${BRANCH}"
RUN git pull
RUN make -f ./Makefile dockerinstall
# The final version should not use this base image, so it needs to have 'cloc' installed.
FROM dajohn/dev-analytics-grimoire-docker-minimal
COPY docker-images/.gitconfig /root/.gitconfig
COPY docker-images/scripts/*.sh /
COPY --from=builder /go/bin/syncdatasources /usr/bin/
COPY --from=builder /go/bin/dads /usr/bin/
COPY --from=builder /go/bin/gitops /usr/bin/
COPY sources/data.zip /data.zip
COPY sources/uuid.py /usr/bin/
COPY sources/gitops.py /usr/bin/
COPY sources/detect-removed-commits.sh /usr/bin/
#COPY sources/sds.yaml /data/sds.txt
COPY docker-images/.vimrc /root/.vimrc
CMD /bin/bash
