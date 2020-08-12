package main

import (
	"fmt"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
)

const (
	deploymentsYAML = "sds-deployment.yaml"
)

type deployment struct {
	Name          string            `yaml:"name"`
	Master        bool              `yaml:"master"`
	Cron          string            `yaml:"cron"`
	CommandPrefix string            `yaml:"command_prefix"`
	CSVPrefix     string            `yaml:"csv_prefix"`
	CronEnv       string            `yaml:"cron_env"`
	Env           map[string]string `yaml:"env"`
}

type environment struct {
	Name        string       `yaml:"name"`
	Deployments []deployment `yaml:"deployments"`
}

type sdsDeployment struct {
	Environments []environment `yaml:"environments"`
}

func deployCrontab(deployEnv string) (err error) {
	data, err := ioutil.ReadFile(deploymentsYAML)
	if err != nil {
		fmt.Printf("Error reading file: %s: %+v\n", deploymentsYAML, err)
		return
	}
	var deployment sdsDeployment
	err = yaml.Unmarshal(data, &deployment)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s: %+v\n", deploymentsYAML, err)
		return
	}
	fmt.Printf("%+v\n", deployment)
	return
}

func main() {
	// */10 * * * * PATH=$PATH:/snap/bin /usr/bin/sds-main.sh
	// SDS_CSV_PREFIX='/root/jobs' SDS_SILENT=1 SDS_TASK_TIMEOUT_SECONDS=43200 SDS_NCPUS_SCALE=2 SDS_SCROLL_SIZE=500 SDS_ES_BULKSIZE=500 SDS_TASKS_SKIP_RE='^sds-cncf-' /usr/bin/sds-cron-task.sh sds_main prod 1>> /tmp/sds_main.log 2>>/tmp/sds_main.err
	// 0 8 * * * PATH=$PATH:/snap/bin /usr/bin/sds-cncf.sh
	// SDS_CSV_PREFIX='/root/cncf_jobs' SDS_NCPUS_SCALE=0.3 SDS_SILENT=1 SDS_TASK_TIMEOUT_SECONDS=43200 SDS_SCROLL_SIZE=1000 SDS_ES_BULKSIZE=1000 SDS_ONLY_P2O=1 SDS_TASKS_RE='^sds-cncf-' /usr/bin/sds-cron-task.sh sds_cncf prod 1>> /tmp/sds_cncf.log 2>>/tmp/sds_cncf.err
	// 0 * * * * PATH=$PATH:/snap/bin SDS_CSV_PREFIX='/root/jobs' SDS_SILENT=1 SDS_NCPUS_SCALE=2 SDS_SCROLL_SIZE=500 SDS_ES_BULKSIZE=500 SDS_TASKS_SKIP_RE='(project1-bugzilla|project1-gerrit|project1-rocketchat|academy-software-foundation-openshadinglanguage-github)' /usr/bin/sds-cron-task.sh sds_main test 1>> /tmp/sds_main.log 2>>/tmp/sds_main.err
	// */10 * * * * PATH=$PATH:/snap/bin SDS_CSV_PREFIX='/root/slow_jobs' SDS_SKIP_REENRICH='jira,gerrit,confluence,bugzilla' SDS_SILENT=1 SDS_ES_BULKSIZE=100 SDS_ONLY_P2O=1 SDS_TASKS_RE='(project1-bugzilla|project1-gerrit|project1-rocketchat|academy-software-foundation-openshadinglanguage-github)' /usr/bin/sds-cron-task.sh sds_slow test 1>> /tmp/sds_slow.log 2>>/tmp/sds_slow.err
	if len(os.Args) < 2 {
		fmt.Printf("Required deployment environment name: [prod|test]\n")
		os.Exit(1)
	}
	err := deployCrontab(os.Args[1])
	if err != nil {
		fmt.Printf("deployCrontab: %+v\n", err)
	}
}
