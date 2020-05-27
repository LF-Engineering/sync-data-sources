#!/bin/bash
if [ -z "$1" ]
then
  echo "$0: you must specify grep pattern to kill"
  exit 1
fi
for pid in `ps -axu | grep p2o | grep "$1" | awk '{ print $2 }'`
do
  echo $pid
  kill $pid
done
