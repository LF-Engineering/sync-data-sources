# How to build grimoire base image for SDS

One command:

- Run `DOCKER_USER=lukaszgryglicki ./build_minimal.sh`.

Manually: 

- Run `docker build -f Dockerfile -t "lukaszgryglicki/dev-analytics-grimoire-docker-minimal" .` to build `dev-analytics-grimoire-docker-minimal` image.
- Run `docker push "lukaszgryglicki/dev-analytics-grimoire-docker-minimal"` to push image to the Docker Hub.
- Run `docker run -it "lukaszgryglicki/dev-analytics-grimoire-docker-minimal" /bin/bash` to test.

# Test image

- Run: `DOCKER_USER=lukaszgryglicki ./test_image.sh`.

