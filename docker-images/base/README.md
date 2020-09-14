# How to build grimoire base image for SDS

One command:

- Run `docker logout && docker login && DOCKER_USER=dajohn ./build_minimal.sh`.

Manually: 

- Run `docker build -f Dockerfile -t "dajohn/dev-analytics-grimoire-docker-minimal" .` to build `dev-analytics-grimoire-docker-minimal` image.
- Run `docker push "dajohn/dev-analytics-grimoire-docker-minimal"` to push image to the Docker Hub.
- Run `docker run -it "dajohn/dev-analytics-grimoire-docker-minimal" /bin/bash` to test.

# Test image

- Run: `DOCKER_USER=dajohn ./test_image.sh`.

