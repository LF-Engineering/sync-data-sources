# sync-data-sources

Single go binary that will manage Grimoire stack data gathering using configuration fixtures from dev-analytics-api


# Docker images

- Use: `DOCKER_USER=docker-user BRANCH=test [API_REPO_PATH="$HOME/dev/LF/dev-analytics-api"] [SKIP_BUILD=1] [SKIP_PUSH=1] ./docker-images/build.sh` to build docker image.
- Use: `DOCKER_USER=docker-user BRANCH=test|prod [PRUNE=1] ./docker-images/remove.sh` to remove docker image locally (remote version is not touched).
- Use: `DOCKER_USER=docker-user BRANCH=test|prod ./docker-images/test_image_docker.sh [command]` to test docker image locally.


# Kubernetes

- Use: `DOCKER_USER=docker-user BRANCH=test|prod ./kubernetes/test_image_kubernetes.sh [command]` to test docker image on kubernetes (without Helm chart).
