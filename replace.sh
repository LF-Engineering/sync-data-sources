#!/bin/bash
if [ -z "$1" ]
then
  echo "You need to provide path as first arument"
  exit 1
fi
if [ -z "$2" ]
then
  echo "You need to provide file name pattern as a second argument"
  exit 1
fi
if [ -z "$3" ]
then
  echo "You need to provide regexp pattern to search for as a third argument"
  exit 1
fi
if [ -z "$4" ]
then
  echo "You need to provide replacement as a fourth argument"
  exit 1
fi
for f in `find "$1" -type f -iname "$2" -not -name "out" -not -path '*.git/*' -exec grep -EHIl "$3" "{}" \;`
do
  echo "$f '$3' -> '$4'"
  sed -i "s/${3}/${4}/g" "${f}" || echo "failed: $f '$3' -> '$4'"
done
