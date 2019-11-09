#!/bin/bash
if [ -z "${GERRIT_USER}" ]
then
  echo -n "Gerrit Password: "
  read GERRIT_USER
fi
perceval gerrit --category review --user "${GERRIT_USER}" gerrit.onosproject.org > .gerrit.onosproject.org.gerrit.log
