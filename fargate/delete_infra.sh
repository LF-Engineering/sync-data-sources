#!/bin/bash
# AP=1 - delete with AP mode
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
