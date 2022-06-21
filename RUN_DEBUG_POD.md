# How to run debug pod on ec2prod

go to ec2prod, then: cd go/src/github.com/LF-Engineering/sync-data-sources/
SH=1 DBG=1 DRY='' ./docker-images/run_sds.sh prod
you are now inside sds debug pod
/fetch.sh to fetch all data
all fixtures are now in /data - move them elsewhere mv /data /d
now mkdir/data - you have empty fixtures directory
copy anything u need to run from /d to /data, like cp /d/xen/unikraft.yaml /data
eventually edit that fixture: vim /data/unikraft.yaml
see example scripts that can be run manually ls /*.sh
especially /debug_run.sh
example command: DA_GIT_FORCE_FULL=1 SDS_SKIP_ES_DATA=1 SDS_SKIP_ES_LOG=1 SDS_DEBUG=2 SDS_CMDDEBUG=2 SDS_ONLY_P2O=1 SDS_SKIP_AFFS=1 SDS_SKIP_DATA='' SDS_DRY_RUN='' DA_AFFS_API_FAIL_FATAL=1 syncdatasources 2>&1 | tee -a /sds.log
MAKE SURE YOU ALWAYS PASS: SDS_ONLY_P2O=1
otherwise SDS will “think” that all fixtures in the ssytem are those you want to sync now, so it will assume everything else needs to be DROPPED!
so ALWAYS pass SDS_ONLY_P2O=1
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
now some useful options:
DA_GIT_FORCE_FULL=1 - to force full sync, not incremental, replace GIT with JIRA etc
Higher debug levels: SDS_DEBUG=2 SDS_CMDDEBUG=2
SDS_SKIP_AFFS=1 - skip affs refresh
SDS_SKIP_DATA=1 - skip data sync (incremental) so if you specify both SDS_SKIP_DATA=1 and SDS_SKIP_AFFS=1 - nothing will happen
SDS_DRY_RUN=1 - run in dry-run mode
DA_AFFS_API_FAIL_FATAL=1 - make any affs API failure fatal
you can see all possible SDS env variable options in here: https://github.com/LF-Engineering/sync-data-sources/blob/master/sources/context.go#L12
