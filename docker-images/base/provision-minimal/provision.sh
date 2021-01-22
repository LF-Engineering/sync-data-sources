#!/usr/bin/env bash
set -eo pipefail
export SETUPTOOLS_USE_DISTUTILS=stdlib
# export PIP_USE_FEATURE=2020-resolver
python3 -m pip install --upgrade setuptools
python3 -m pip install --upgrade pip
python3 -m pip install emoji
python3 -m pip install --upgrade requests

repos=(grimoirelab-elk grimoirelab-sortinghat grimoirelab-kingarthur grimoirelab-perceval)

for r in "${repos[@]}"; do 
  echo "INSTALLING ${r}"
  python3 -m pip install -e "${REPOS_DIR}/$r";
  # pip3 install -e "${REPOS_DIR}/$r";
done

python3 -m pip install geopy==2.0.0
python3 -m pip install six==1.12.0
python3 -m pip install "feedparser>=5.1.3,<6.0.0"
python3 -m pip install "sqlalchemy<1.4,>=1.2"
python3 -m pip install PyMySQL==0.9.3
python3 -m pip install PyJWT>=1.7.1
python3 -m pip install urllib3==1.24.3

cp repos/cloc/cloc /usr/bin/
