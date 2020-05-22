#!/bin/bash
# AP=1 - list with AP mode (list access point too)
# LG=1 - list log groups and log streams
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
# Clusters
echo 'Clusters:'
./fargate/list_clusters.sh
# Log groups and streams
if [ ! -z "${LG}" ]
then
  echo 'Log groups:'
  ./fargate/list_log_groups.sh
  echo 'Log streams for sds-logs:'
  ./fargate/list_log_streams.sh "sds-logs"
fi
# Role: task role and role policy for task execution
echo 'Role:'
./fargate/list_roles.sh
# VPC: main VPC
echo 'VPC:'
./fargate/list_vpcs.sh
# IGW: internet gateway and route tables
echo 'Gateway:'
./fargate/list_igws.sh
# Subnet: subnet and route table associations
echo 'Subnet:'
./fargate/list_subnets.sh
# SGs: main SG, EFS MT SG (with their ingress rules)
echo 'Security groups:'
./fargate/list_security_groups.sh
# EFS: mount target, access point (in AP mode), filesystem
echo 'EFS:'
./fargate/list_efs.sh
