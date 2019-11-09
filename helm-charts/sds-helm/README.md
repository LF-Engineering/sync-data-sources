# sds-helm

Helm chart for sync-data-sources tool (it loads all `dev-analytics-api` fixtures and precesses their data sources using Grimoire stack tools).


# Usage

Please provide secret values for each file in `./secrets/*.secret.example` saving it as `./secrets/*.secret` or specify them from the command line.

Please note that `vim` automatically adds new line to all text files, to remove it run `truncate -s -1` on a saved file.

List of secrets:
- File `secrets/SH_USER.secret` or --set `shUser=...` setup MariaDB admin user name.
- File `secrets/SH_HOST.env.secret` or --set `shHost=...` setup MariaDB host name.
- File `secrets/SH_PORT.secret` or --set `shPort=...` setup MariaDB port.
- File `secrets/SH_PASS.env.secret` or --set `shPass=...` setup MariaDB password.
- File `secrets/SH_DB.secret` or --set `shDB=...` setup MariaDB database.
- File `secrets/ES_URL.secret` or --set `esURL=...` setup ElasticSearch connection.
- File `secrets/ZIPPASS.secret` or --set `zipPass=...` fixtures unzip password (fixtures contain sensitive information so they're stored in password protected zip file inside the docker image).

To install:
- `helm install sds ./sds-helm --set deployEnv=test|prod`.

To upgrade:
- `helm upgrade sds ./sds-helm`.

You can install only selected templates, see `values.yaml` for detalis (refer to `skipXYZ` variables in comments), example:
- `helm install --dry-run --debug --generate-name ./sds-helm --set deployEnv=test,skipSecrets=1,skipCron=1,skipNamespace=1,skipPV=1`.

Please note variables commented out in `./sds-helm/values.yaml`. You can either uncomment them or pass their values via `--set variable=name`.

Import company affiliations from cncf/devstats into GrimoireLab Sorting Hat database.

Other environment parameters:

- `SDS_MAXRETRY`/`sdsMaxRetry` - Try to run grimoire stack (perceval, p2o.py etc) that many times before reporting failure, default 3
- `SDS_DEBUG`/`sdsDebug` - Debug level: 0-no, 1-info, 2-verbose.
- `SDS_CMDDEBUG`/`sdsCmdDebug` - Commands execution Debug level: 0-no, 1-only output commands, 2-output commands and their output, 3-output full environment as well, default.
- `SDS_ST`/`sdsST` - force using single-threaded version.
- `SDS_NCPUS`/`sdsNCPUs` - set to override number of CPUs to run, this overwrites `SDS_ST`, default 0 (which means autodetect).
- `SDS_CTXOUT`/`sdsCtxOut` - output all context data (configuration struct).
- `SDS_SKIPTIME`/`sdsSkipTime` - do not output time with log messages.

# Deploy on LF infra

- Install: `[DBG=1] ./setup.sh test|prod`.
- Shell into debug pod (only when installed with `DBG=1`): `pod_shell.sh test sds sds-debug`.
- Unnstall: `./delete.sh test|prod`.
