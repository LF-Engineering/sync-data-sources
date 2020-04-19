# sync-data-sources

Single go binary that will manage Grimoire stack data gathering using configuration fixtures from dev-analytics-api


# Docker images

- First you will need a base image containing grimoire stack tools needed, follow instructions on `LF-Engineering/dev-analytics-grimoire-docker` to build minimal image.
- Use: `DOCKER_USER=docker-user BRANCH=test [API_REPO_PATH="$HOME/dev/LF/dev-analytics-api"] [SKIP_BUILD=1] [SKIP_PUSH=1] [SKIP_PULL=1] [PRUNE=1] ./docker-images/build.sh` to build docker image.
- Use: `DOCKER_USER=docker-user [SKIP_BUILD=1] [SKIP_PUSH=1] [PRUNE=1] ./docker-images/build-validate.sh` to build docker image file (validation image),
- Use: `DOCKER_USER=docker-user BRANCH=test|prod [PRUNE=1] ./docker-images/remove.sh` to remove docker image locally (remote version is not touched).
- Use: `` DOCKER_USER=docker-user BRANCH=test|prod [REPO_ACCESS="`cat repo_access.secret`"] ./docker-images/test_image_docker.sh [command] `` to test docker image locally. Then inside the container run: `./run.sh`.
- Use: `` DOCKER_USER=docker-user BRANCH=commit_SHA [REPO_ACCESS="`cat repo_access.secret`"] ./docker-images/test_validate_image_docker.sh [command] `` to test docker image locally. Then inside the container run: `./run.sh`.


# Manual docker run example

- Get SortingHat DB endpoint: `prodk.sh -n mariadb get svc mariadb-service-rw`.
- Get SortingHat DB credentials: `for f in helm-charts/sds-helm/sds-helm/secrets/SH_*.prod.secret; do echo -n "$f: "; cat $f; echo ""; done`.
- Get other env variables: `prodk.sh -n sds edit cj sds-0`.
- Finally just run `[EXEC=1] ./docker-images/manual_docker.sh prod`. If you specify EXEC=1 you will get bash instead of running SDS, so you can call `./run.sh` manually.
- Validation `[EXEC=1] ./docker-images/manual_docker_validate.sh commit_sha`. If you specify EXEC=1 you will get bash instead of running SDS, so you can call `./run.sh` manually.
- To see environment inside the container: `clear; env | sort | grep 'SDS\|SH_'`.
- Send info signal `` kill -SIGUSR1 `ps -ax | grep syncdatasources | head -n 1 | awk '{print $1}'` ``.
- To shell into the running SDS: `./docker-images/shell_running_sds.sh`.


# Kubernetes

- Use: `DOCKER_USER=docker-user BRANCH=test|prod ./kubernetes/test_image_kubernetes.sh [command]` to test docker image on kubernetes (without Helm chart). Then inside the container run: `./run.sh`.


# Locally

- Use: `cd sources; make; ./dry_run.sh prod` even if you don't have correct `p2o.py` stack installed.


# Helm

- Go to `helm-charts/sds-helm` and follow instructions from `README.md` file in that derectory. Helm chart contains everything needed to run entire stack.
- Note that `zippass.secret` and `helm-charts/sds-helm/sds-helm/secrets/ZIPPASS.secret` files should have the same contents.
- See documentation [here](https://github.com/LF-Engineering/sync-data-sources/blob/master/helm-charts/sds-helm/README.md).

# In short

- Install: `[NODES=n] [NS=sds] ./setup.sh test|prod`.
- Unnstall: `[NS=sds] ./delete.sh test|prod`.
- Other example (with external ES): `` NODES=2 NS=sds-ext FLAGS="esURL=\"`cat sds-helm/secrets/ES_URL_external.secret`\",pvSize=30Gi" ./setup.sh test ``.

# Debug

- If not installed with the Helm chart (which is the default), for the `test` env do: `cd helm-charts/sds-helm/`, `[NODES=n] [NS=sds] ./debug.sh test`, `pod_shell.sh test sds sds-debug-0` to get a shell inside `sds` deployment. Then `./run.sh`.
- When done `exit`, then copy CSV: `testk.sh -n sds cp sds-debug-0:root/.perceval/tasks_0_1.csv tasks_test.csv`, finally delete debug pod: `[NS=sds] ./debug_delete.sh test`.
