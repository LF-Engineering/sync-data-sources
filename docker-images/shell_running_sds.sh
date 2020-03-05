#!/bin/bash
docker exec -it `docker container ls | grep '"./run.sh"' | head -n 1 | awk '{print $1}'` /bin/bash
