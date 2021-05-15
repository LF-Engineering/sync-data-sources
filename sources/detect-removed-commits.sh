#!/bin/bash
#for f in `find .git/objects/??/ -type f | sed 's/\(.*\)\/\([[:xdigit:]]\{2\}\)\/\([[:xdigit:]]\+\)$/\2\3/g'`
declare -A glog
declare -A gfile
set -o pipefail
set -e
if [ -z "`git rev-list -n 1 --all`" ]
then
  exit 0
fi
allcommits=`git cat-file --unordered --batch-all-objects --buffer --batch-check | grep ' commit ' | awk '{print $1}'`
for f in $allcommits
do
  gfile[$f]=1
done
commits=`git rev-list --all --remotes`
for f in $commits
do
  glog[$f]=1
done
missing=''
for f in "${!gfile[@]}"
do
  got=${glog[$f]}
  if [ ! "$got" = "1" ]
  then
    if [ -z "${missing}" ]
    then
      missing="$f"
    else
      missing="${missing} ${f}"
    fi
  fi
done
if [ ! -z "${missing}" ]
then
  echo -n "${missing}"
fi
