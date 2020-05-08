# sds-helm

Helm chart for sync-data-sources tool (it loads all `dev-analytics-api` fixtures and precesses their data sources using Grimoire stack tools).


# Usage

Please provide secret values for each file in `./secrets/*.secret.example` saving it as `./secrets/*.secret` or specify them from the command line.

Please note that `vim` automatically adds new line to all text files, to remove it run `truncate -s -1` on a saved file.

List of secrets:
- File `secrets/SH_USER.secret` or --set `shUser=...` setup MariaDB admin user name.
- File `secrets/SH_HOST.env.secret` or --set `shHost=...` setup MariaDB host name.
- File `secrets/SH_PASS.env.secret` or --set `shPass=...` setup MariaDB password.
- File `secrets/SH_DB.secret` or --set `shDB=...` setup MariaDB database.
- File `secrets/ES_URL.secret` or --set `esURL=...` setup ElasticSearch connection.
- File `secrets/SSAW_URL.secret` or --set `ssawURL=...` setup SSAW endpoint.
- File `secrets/ZIPPASS.secret` or --set `zipPass=...` fixtures unzip password (fixtures contain sensitive information so they're stored in password protected zip file inside the docker image).

To install:
- `helm install sds ./sds-helm --set deployEnv=test|prod`.

To upgrade:
- `helm upgrade sds ./sds-helm`.

You can install only selected templates, see `values.yaml` for detalis (refer to `skipXYZ` variables in comments), example:
- `helm install --dry-run --debug --generate-name ./sds-helm --set deployEnv=test,skipSecrets=1,skipCron=1,skipNamespace=1,skipPV=1`.
- Install debug pod: `helm install sds-debug ./sds-helm --set deployEnv=test,skipSecrets=1,skipCron=1,skipNamespace=1,skipPV=1,debugPod=1`.

Please note variables commented out in `./sds-helm/values.yaml`. You can either uncomment them or pass their values via `--set variable=name`.

Import company affiliations from cncf/devstats into GrimoireLab Sorting Hat database.

Other environment parameters:

- `SDS_MAXRETRY`/`sdsMaxRetry` - Try to run grimoire stack (perceval, p2o.py etc) that many times before reporting failure, default 3 (1 original and 2 more attempts).
- `SDS_DEBUG`/`sdsDebug` - Debug level: 0-no, 1-info, 2-verbose.
- `SDS_CMDDEBUG`/`sdsCmdDebug` - Commands execution Debug level: 0-no, 1-only output commands, 2-output commands and their output, 3-output full environment as well, default.
- `SDS_ST`/`sdsST` - force using single-threaded version.
- `SDS_NCPUS`/`sdsNCPUs` - set to override number of CPUs to run, this overwrites `SDS_ST`, default 0 (which means autodetect).
- `SDS_CTXOUT`/`sdsCtxOut` - output all context data (configuration struct).
- `SDS_SKIPTIME`/`sdsSkipTime` - do not output time with log messages.
- `SDS_ES_BULKSIZE`/`esBulkSize` - ElasticSearch bulk size when enriching data (default is 1000).
- `SDS_SCROLL_SIZE`/`scrollSize` - ElasticSearch scroll size when processing data (default is 1000).
- `SDS_SCROLL_WAIT`/`scrollWait` - time to wait for ElasticSearch scroll to become available (when all scrolls are used), default 900 (seconds).
- `SDS_DRY_RUN`/`dryRun` - Run in dry-run mode, do not execute any grimoire stack command, report success instead.
- `SDS_DRY_RUN_CODE`/`dryRunCode` - When in dry-run mode, set fake grimoire command result exit code.
- `SDS_DRY_RUN_SECONDS`/`dryRunSeconds` - When in dry-run mode, set fake grimoire command running time in seconds.
- `SDS_TIMEOUT_SECONDS`/`timeoutSeconds` - if program didn't finished before this timeout (in seconds), finish it with exit code 2. Default 47h 45min (171900s).
- `SDS_N_LONGEST`/`nLongest` - number of longest running tasks to display in stats, default 10.
- `SDS_STRIP_ERROR_SIZE`/`stripErrorSize` - error messages longer that this value will be stripped by half of this value from beginning and from end
- `SDS_SKIP_SH`/`skipSH` - do not use SortingHat.
- `SDS_SILENT`/`silent` - do not pass `-g` (debug) flag to `p2o.py` calls, makes output a lot less verbose.
- `SDS_SKIP_DATA`/`skipData` - do not run incremental data sync.
- `SDS_SKIP_AFFS`/`skipAffs` - do not re-enrich historical affiliations data.
- `SDS_SKIP_ALIASES`/`skipAliases` - do not create index aliases, do not attempt to drop unused aliases.
- `SDS_NO_MULTI_ALIASES`/`noMultiAliases` - alias names must be unique, so every alias can only point to a single index. If not set then single alias can point to multiple indices.
- `SDS_CLEANUP_ALIASES`/`cleanupAliases` - drop aliases before creating them, this can be used to clean existing aliases from some orphaned/no longer needed indexes.
- `SDS_SKIP_DROP_UNUSED`/`skipDropUnused` - do not drop unused indexes/aliases.
- `SDS_SKIP_ES_DATA`/`skipEsData`  do not process "sdsdata" index at all (SDS state saved in ES).
- `SDS_SKIP_CHECK_FREQ`/`skipCheckFreq` - skip check sync frequency.
- `SDS_MAX_DELETE_TRIALS`/`maxDeleteTrials` - set maximum retries for delete by query.
- `SDS_SKIP_ES_LOG`/`skipEsLog`  do not store SDS logs in ES index "sdslog".

# Deploy on LF infra

- Install: `[DBG=1] [DRY=1] [ES_BULK_SIZE=10000] [NODES=n] [NS=sds] [FLAGS=...] ./setup.sh test|prod`.
- Shell into debug pod (only when installed with `DBG=1`): `pod_shell.sh test sds sds-debug`.
- Inside the pod you can for example run: `p2o.py --enrich --index sds_raw --index-enrich sds_enriched -e "http://$SDS_ES_URL" --debug --db-host "${SH_HOST}" --db-sortinghat "${SH_DB}" --db-user "${SH_USER}" --db-password "${SH_PASS}" github cncf devstats --category issue -t key1 key2 --sleep-for-rate`.
- Unnstall: `[NS=sds] ./delete.sh test|prod`.
- To have more than one `SDS` running on the same cluster use: `NS=sds2`, specify any namespace name other than default `sds`.


# Debug pod

- If not installed with the Helm chart (which is the default), for the `test` env do: `cd helm-charts/sds-helm/`, `[NODES=n] [NS=sds] [FLAGS=...] ./debug.sh test`, `pod_shell.sh test sds sds-debug-0` to get a shell inside `sds` deployment. Then `./run.sh`.
- When done `exit`, then copy CSV: `testk.sh -n sds cp sds-debug-0:root/.perceval/tasks_0_1.csv tasks_test.csv`, finally delete debug pod: `[NS=sds] ./debug_delete.sh test`.


# Deploying on multiple nodes

You can use multiple nodes deployment, you need to know how many nodes you have, pods will use node antiaffinity to ensure they're put on separate nodes, also PVs will be per-node because we need a fast `ReadWriteOnce` access mode.

- Use `SDS_NODE_HASH`/`nodeHash` to set multi node deployment (it will use hash function to distribute work across nodes without duplicating).
- Use `SDS_NODE_NUM`/`nodeNum` to specify number of nodes available. each will get about `1/N` of work.
- Use `SDS_NODE_IDX`/`nodeIdx` to specify which node index you're now deploying - you need to make N deployments, each for different node.
- You need to add `NODES=n` to `setup.sh` script call.
