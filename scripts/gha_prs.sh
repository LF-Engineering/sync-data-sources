#!/bin/bash
if [ -z "$1" ]
then
  echo "$0: you need to specify repo name as a 1st argument: org/name"
  exit 1
fi
if [ -z "$2" ]
then
  echo "$0: you need to specify GHA data as a 2nd argument: YYYY-MM-DD-H"
  exit 2
fi
fns=${2}
if [ ${#2} -le "10" ]
then
  fns="${fns}-0 "
  for i in {1..3}
  do
    fns="${fns} ${2}-${i}"
  done
fi
declare -A prs
for f in $fns
do
  fn="${f}.json"
  zfn="${fn}.gz"
  echo -n "$f -> $zfn -> $fn -> "
  if ( [ ! -f "${zfn}" ] && [ ! -f "${fn}" ] )
  then
    wget "http://data.gharchive.org/${zfn}" 1>/dev/null 2>/dev/null || exit 3
  fi
  if [ ! -f "${fn}" ]
  then
    gzip -d "${zfn}" || exit 4
  fi
  pfn="processed_${fn}"
  if [ ! -f "${pfn}" ]
  then
    sed '1s/^/[/;$!s/$/,/;$s/$/]/' "${fn}" > "${pfn}" || exit 5
  fi
  echo -n "$pfn: "
  for r in `cat "${pfn}" | jq ".[] | select(.repo.name == \"${1}\") | .payload.pull_request.number" | sort | uniq`
  do
    if [ ! "$r" = "null" ]
    then
      prs[$r]=1
      echo -n "$r, "
    fi
  done
  echo ''
done
echo "${!prs[@]}" | tr ' ' '\n' | sort -n | tr '\n' ', '
echo ''
