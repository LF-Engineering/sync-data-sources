# sync-data-sources

Single go binary that will manage Grimoire stack data gathering using configuration fixtures from dev-analytics-api


# Docker images

- Use: `DOCKER_USER=docker-user BRANCH=test ./docker-images/build.sh` to build docker image.
- Use: `DOCKER_USER=docker-user BRANCH=test|prod [PRUNE=1] ./docker-images/remove.sh` to remove docker image locally (remote version is not touched).
