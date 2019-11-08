#!/bin/sh
if [ -z "$ZIPPASS" ]
then
  echo "$0: you have to specify unzip password via ZIPPASS=..."
  exit 1
fi
unzip -P "$ZIPPASS" data.zip > /dev/null
