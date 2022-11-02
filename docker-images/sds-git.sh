#/bin/bash
SDS_ONLY_P2O=1 SDS_CSV_PREFIX=/root/sds-prod-main SDS_SILENT='1' SDS_NCPUS_SCALE='1.5' SDS_TASKS_SKIP_RE='^sds-(lfn-odl-gerrit|open-mainframe-project|accord|acrn|allseen|alljoyn|act|agl|bitcoin-protocol|cc|CC|cephfoundation|chaoss|ojsf-chassis|chips|cip|clusterduck|cni|caf|ccc|cdf|cord-datacenter|cord-central-office-re-architected-as-a-datacenter|cregit|datapractices|dent|desktopwg|easycla|elisa|finops|finos|fossbazaar|frr|gql|internet-security-research-group|interuss|iovisor|janusgraph|kernelci|openkinetic|laminas|lets-encrypt|lf-asia-llc|lfph|lsb|linuxboot|llvmlinux|ltsi|lvfs|mapzen|mojaglobal|newticity|onf|O3DE|o3de|a11y|open-alliance-for-cloud-adoption|opencontainers|open-network-operating-system-onos|osecc|osapa|ova|open-voice-network|ovs|openbel|openbmc|openchain|OEEW|oeew|openfabrics-inc|openhpc|ojsf|openmama|openmessaging|openpowerfoundation|opensds|osquery|patentcommons|quartermaster|rcons|real-time-linux|redteam|rtdb|riscv|risc-v-international|sel4|soda-foundation|sof|tlf|openssf|reactive|tizen|ucf|korg|cncf)-' SDS_DATASOURCES_RE='^git' SDS_TASK_TIMEOUT_SECONDS='43200' SDS_SCROLL_SIZE='500' SDS_SKIP_SSAW='1' SDS_ES_BULKSIZE='500' DA_GIT_ALLOW_FAIL='1' DA_AFFS_API_FAIL_FATAL='1' /usr/bin/sds-cron-task.sh sds-prod-git prod 1>> /tmp/sds-prod-git.log 2>>/tmp/sds-prod-git.err
