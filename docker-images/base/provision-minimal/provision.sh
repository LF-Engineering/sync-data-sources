#!/usr/bin/env bash
set -eo pipefail
export SETUPTOOLS_USE_DISTUTILS=stdlib
# export PIP_USE_FEATURE=2020-resolver
pip3 install --upgrade setuptools
pip3 install --upgrade "pip<=20.2.4"
pip3 install emoji
pip3 install --upgrade requests

repos=(grimoirelab-perceval grimoirelab-elk grimoirelab-sortinghat grimoirelab-kingarthur)

for r in "${repos[@]}"; do 
  echo "INSTALLING ${r}"
  # python3 -m pip install -e "${REPOS_DIR}/$r";
  pip3 install -e "${REPOS_DIR}/$r";
done

pip3 install geopy==2.0.0
pip3 install six==1.12.0
pip3 install "feedparser>=5.1.3,<6.0.0"
pip3 install "sqlalchemy<1.4,>=1.2"
pip3 install PyMySQL==0.9.3

cp repos/cloc/cloc /usr/bin/
