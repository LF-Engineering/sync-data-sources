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
if [ -z "${3}" ]
then
  echo "$0: you need to specify service name as a third argument"
  exit 4
fi
aws ecs delete-service --cluster "${2}-${1}" --service "${3}-${1}" --force
