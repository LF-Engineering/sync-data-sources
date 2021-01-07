FROM debian:10-slim
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
    apt-get -y install --no-install-recommends apt-utils bash locales gcc cloc git git-core jq procps tree ccze psmisc python python3 \
    python3-setuptools python3-distlib python3-wheel python3-venv python3-dev python3-gdbm python3-requests python3-pip \
    build-essential libssl-dev libffi-dev mariadb-client vim unzip curl wget openssh-client && \
    apt-get clean && \
    find /var/lib/apt/lists -type f -delete
RUN sed -i -e 's/# en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen && \
    echo 'LANG="en_US.UTF-8"'>/etc/default/locale && \
    dpkg-reconfigure --frontend=noninteractive locales && \
    update-locale LANG=en_US.UTF-8
ENV REPOS_DIR=/repos
ADD repos ${REPOS_DIR}
ADD provision-minimal /provision
ENV LANG en_US.UTF-8
ENV LANGUAGE en_US:en
ENV LC_ALL en_US.UTF-8
ENV LANG C.UTF-8
RUN provision/provision.sh
