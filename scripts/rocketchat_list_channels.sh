#!/bin/bash
curl -s -H "X-Auth-Token: aaa_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb-cccc" -H "X-User-Id: ddddddddddddddddd" 'https://chat.hyperledger.org/api/v1/channels.list?fields=%7b%22name%22%3a1%7d&count=10&offset=10' | jq '.channels[].name' | sort
