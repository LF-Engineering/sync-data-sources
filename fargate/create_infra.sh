#!/bin/bash
# AP=1 - create with AP mode (access point too)
# LG=1 - create log group (otherwise task does it at 1st execution)
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
# Clusters
echo 'Clusters:'
./fargate/create_cluster.sh test sds-cluster || exit 2
./fargate/create_cluster.sh prod sds-cluster || exit 3
# Log groups and streams
if [ ! -z "${LG}" ]
then
  echo 'Log group:'
  ./fargate/create_log_group.sh || exit 4
fi
# Role: task role and role policy for task execution
echo 'Role:'
SS=1 ./fargate/create_role.sh || exit 5
# VPC: main VPC
echo 'VPC:'
SS=1 ./fargate/create_vpc.sh || exit 6
# IGW: internet gateway and route tables
echo 'Gateway:'
./fargate/create_igw.sh || exit 7
# Subnet: subnet and route table associations
echo 'Subnet:'
SS=1 ./fargate/create_subnet.sh || exit 8

echo 'AllOK'
exit 111
# SGs: main SG, EFS MT SG (with their ingress rules)
echo 'Security groups:'
./fargate/list_security_groups.sh
# EFS: mount target, access point (in AP mode), filesystem
echo 'EFS:'
./fargate/list_efs.sh
