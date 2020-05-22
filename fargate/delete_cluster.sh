#!/bin/bash
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
if [ -z "${1}" ]
then
  echo "$0: you need to specify env as a first argument: test|prod"
  exit 2
fi
if [ -z "${2}" ]
then
  echo "$0: you need to specify cluster name as a second argument"
  exit 3
fi
clustername=`aws ecs delete-cluster --cluster "${2}-${1}" 2>/dev/null | jq -r '.cluster.clusterName'`
if [ ! -z "${clustername}" ]
then
  echo "Deleted cluster ${clustername}"
else
  echo 'No cluster to delete'
fi
