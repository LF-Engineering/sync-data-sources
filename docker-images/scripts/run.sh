#!/bin/sh
if [ -z "$REPO_ACCESS" ]
then
  ./unzip.sh && syncdatasources
else
  ./fetch.sh && syncdatasources
fi
