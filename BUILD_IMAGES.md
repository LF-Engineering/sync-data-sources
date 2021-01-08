# Build images workflow

The full workflow is to start in SDS directory and then:
- `cd ./docker-images/base/`.
- `./collect_and_build.sh`.
- `docker build -f Dockerfile -t "dajohn/dev-analytics-grimoire-docker-minimal" .`.
- Do not push yet! Check if p2o.py works inside the image (which is not the case very often due to internal incompatibilities withing the p2o stack):
- `docker run -it "dajohn/dev-analytics-grimoire-docker-minimal" p2o.py`.
- If p2o.py works OK then:
- `docker push "dajohn/dev-analytics-grimoire-docker-minimal"`.
- When this is finished back to SDS main directory: `cd ../../`.
- Build the `validate` image: `DOCKER_USER=dajohn PRUNE=1 ./docker-images/build-validate.sh`.
- Build the `test` image: `DOCKER_USER=dajohn BRANCH=test PRUNE=1 ./docker-images/build.sh`.
- Check if it works OK: `SH=1 DBG=1 DRY=1 ./docker-images/run_sds.sh test` - this starts SDS in dry-run mode.
- While inside the SDS run: `/fetch.sh && syncdatasources`.
- Build the `prod` image (only when test image is tested and `test` SDS finished at least one normal sync): `DOCKER_USER=dajohn BRANCH=prod PRUNE=1 ./docker-images/build.sh`.
- Check if it works OK: `SH=1 DBG=1 DRY=1 ./docker-images/run_sds.sh prod` - this starts SDS in dry-run mode.
- While inside the SDS run: `/fetch.sh && syncdatasources`.

