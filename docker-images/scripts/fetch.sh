#!/bin/bash
git clone $REPO_ACCESS || exit 1
cd dev-analytics-api || exit 2
git checkout $BRANCH || exit 3
git pull || exit 4
cp -R app/services/lf/bootstrap/fixtures/ /data || exit 5
cd .. && rm -rf dev-analytics-api
