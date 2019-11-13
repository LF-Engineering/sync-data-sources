# sync-data-sources

Single go binary that will manage Grimoire stack data gathering using configuration fixtures from dev-analytics-api


# Docker images

- First you will need a base image containing grimoire stack tools needed, follow instructions on `LF-Engineering/dev-analytics-grimoire-docker` to build minimal image.
- Use: `DOCKER_USER=docker-user BRANCH=test [API_REPO_PATH="$HOME/dev/LF/dev-analytics-api"] [SKIP_BUILD=1] [SKIP_PUSH=1] [PRUNE=1] ./docker-images/build.sh` to build docker image.
- Use: `DOCKER_USER=docker-user BRANCH=test|prod [PRUNE=1] ./docker-images/remove.sh` to remove docker image locally (remote version is not touched).
- Use: `DOCKER_USER=docker-user BRANCH=test|prod ./docker-images/test_image_docker.sh [command]` to test docker image locally. Then inside the container run: `./run.sh`.


# Kubernetes

- Use: `DOCKER_USER=docker-user BRANCH=test|prod ./kubernetes/test_image_kubernetes.sh [command]` to test docker image on kubernetes (without Helm chart). Then inside the container run: `./run.sh`.


# Helm

- Go to `helm-charts/sds-helm` and follow instructions from `README.md` file in that derectory. helm chart contains everything needed to run entire stack.
- Note that `zippass.secret` and `helm-charts/sds-helm/sds-helm/secrets/ZIPPASS.secret` files should have the same contents.
- See documentation [here](https://github.com/LF-Engineering/sync-data-sources/blob/master/helm-charts/sds-helm/README.md).

In short:

- Install: `./setup.sh test|prod`.
- Unnstall: `./delete.sh test|prod`.
