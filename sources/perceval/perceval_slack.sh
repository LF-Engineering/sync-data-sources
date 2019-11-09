#!/bin/bash
if [ -z "${API_TOKEN}" ]
then
  echo -n "Slack legacy API token: "
  read -s API_TOKEN
fi
# random channel
perceval slack C095YQBM2 --category message -t "${API_TOKEN}" >> .onosproject.slack.com.log
# general channel
perceval slack C095YQBLL --category message -t "${API_TOKEN}" > .onosproject.slack.com.log
