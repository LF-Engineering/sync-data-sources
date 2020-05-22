#!/bin/bash
# AP=1 - delete with AP mode (access point gets deleted too)
# LG=1 - delete log group
if [ -z "${AWS_PROFILE}" ]
then
  echo "$0: you need to specify AWS_PROFILE=..."
  exit 1
fi
# EFS: mount target, access point (in AP mode), filesystem
./fargate/delete_efs.sh
# SGs: main SG, EFS MT SG (with their ingress rules)
./fargate/delete_security_groups.sh
# Subnet: subnet and route table associations
./fargate/delete_subnet.sh
# IGW: internet gateway and route tables
./fargate/delete_igw.sh
# VPC: delete main VPC
./fargate/delete_vpc.sh
# Role: task role and role policy for task execution
./fargate/delete_role.sh
# Logs: log group
if [ ! -z "${LG}" ]
then
  ./fargate/delete_log_group.sh
fi
# Clusters: test & prod
./fargate/delete_cluster.sh test sds-cluster
./fargate/delete_cluster.sh prod sds-cluster
