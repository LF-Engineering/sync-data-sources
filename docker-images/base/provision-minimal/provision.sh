#!/usr/bin/env bash

set -eo pipefail

pip3 install --upgrade setuptools
pip3 install --upgrade pip
pip3 install emoji

repos=(grimoirelab-perceval grimoirelab-elk grimoirelab-sortinghat grimoirelab-kingarthur)

for r in "${repos[@]}"; do 
  echo "INSTALLING ${r}"
  # python3 -m pip install -e "${REPOS_DIR}/$r";
  pip3 install -e "${REPOS_DIR}/$r";
done

pip3 install geopy==2.0.0
