#!/bin/bash
if [ -z "$1" ]
then
  echo "$0: you need to specify GHA data as a 1st argument: YYYY-MM-DD-H"
  exit 1
fi
fn="${1}.json"
zfn="${fn}.gz"
if ( [ ! -f "${zfn}" ] && [ ! -f "${fn}" ] )
then
  wget "http://data.gharchive.org/${zfn}" || exit 2
fi
if [ ! -f "${fn}" ]
then
  gzip -d "${zfn}"
fi
ls -l "${fn}"
