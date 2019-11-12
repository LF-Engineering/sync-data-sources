#!/bin/bash
pid=`ps -a | grep syncdatasources | head -n 1 | awk '{print $1}'`
if [ ! -z "$pid" ]
then
  kill -SIGUSR1 "$pid"
fi
