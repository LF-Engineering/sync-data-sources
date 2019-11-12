#!/bin/bash
pid=`ps -aux | grep syncdatasources | head -n 1 | awk '{print $2}'`
if [ ! -z "$pid" ]
then
  kill -SIGUSR1 "$pid"
fi
