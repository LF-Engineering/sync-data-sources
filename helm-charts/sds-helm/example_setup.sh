#!/bin/bash
DBG=1 NODES=2 ES_BULK_SIZE=100 DRY=1 FLAGS='sdsNCPUsScale=2.0,scrollSize=100,skipMerge=1,silent=1' ./setup.sh test
