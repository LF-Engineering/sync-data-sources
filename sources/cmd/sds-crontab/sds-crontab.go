package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	lib "github.com/LF-Engineering/sync-data-sources/sources"
	yaml "gopkg.in/yaml.v2"
)

const (
	deploymentsYAML = "sds-deployment.yaml"
)

type deployment struct {
	Name          string            `yaml:"name"`
	Cron          string            `yaml:"cron"`
	CronEnv       string            `yaml:"cron_env"`
	CommandPrefix string            `yaml:"command_prefix"`
	Master        bool              `yaml:"master"`
	CSVPrefix     string            `yaml:"csv_prefix"`
	Env           map[string]string `yaml:"env"`
}

type environment struct {
	Name        string       `yaml:"name"`
	Deployments []deployment `yaml:"deployments"`
}

type sdsDeployment struct {
	Environments []environment `yaml:"environments"`
}

func crontabDeployments(ctx *lib.Ctx, deps []deployment) (err error) {
	// */10 * * * * PATH=$PATH:/snap/bin /usr/bin/sds-main.sh
	// SDS_CSV_PREFIX='/root/jobs' SDS_SILENT=1 SDS_TASK_TIMEOUT_SECONDS=43200 SDS_NCPUS_SCALE=2 SDS_SCROLL_SIZE=500 SDS_ES_BULKSIZE=500 SDS_TASKS_SKIP_RE='^sds-cncf-' /usr/bin/sds-cron-task.sh sds_main prod 1>> /tmp/sds_main.log 2>>/tmp/sds_main.err
	// 0 8 * * * PATH=$PATH:/snap/bin /usr/bin/sds-cncf.sh
	// SDS_CSV_PREFIX='/root/cncf_jobs' SDS_NCPUS_SCALE=0.3 SDS_SILENT=1 SDS_TASK_TIMEOUT_SECONDS=43200 SDS_SCROLL_SIZE=1000 SDS_ES_BULKSIZE=1000 SDS_ONLY_P2O=1 SDS_TASKS_RE='^sds-cncf-' /usr/bin/sds-cron-task.sh sds_cncf prod 1>> /tmp/sds_cncf.log 2>>/tmp/sds_cncf.err
	// 0 * * * * PATH=$PATH:/snap/bin SDS_CSV_PREFIX='/root/jobs' SDS_SILENT=1 SDS_NCPUS_SCALE=2 SDS_SCROLL_SIZE=500 SDS_ES_BULKSIZE=500 SDS_TASKS_SKIP_RE='(project1-bugzilla|project1-gerrit|project1-rocketchat|academy-software-foundation-openshadinglanguage-github)' /usr/bin/sds-cron-task.sh sds_main test 1>> /tmp/sds_main.log 2>>/tmp/sds_main.err
	// */10 * * * * PATH=$PATH:/snap/bin SDS_CSV_PREFIX='/root/slow_jobs' SDS_SKIP_REENRICH='jira,gerrit,confluence,bugzilla' SDS_SILENT=1 SDS_ES_BULKSIZE=100 SDS_ONLY_P2O=1 SDS_TASKS_RE='(project1-bugzilla|project1-gerrit|project1-rocketchat|academy-software-foundation-openshadinglanguage-github)' /usr/bin/sds-cron-task.sh sds_slow test 1>> /tmp/sds_slow.log 2>>/tmp/sds_slow.err
	// fmt.Printf("%+v\n", deps)
	reS := "("
	for _, d := range deps {
		reS += "sds-" + d.Name + "|"
	}
	reS = reS[0:len(reS)-1] + ")"
	re := regexp.MustCompile(reS)
	ctx.ExecOutput = true
	res, err := lib.ExecCommand(ctx, []string{"crontab", "-l"}, nil, nil)
	if err != nil {
		return
	}
	lines := []string{}
	allLines := strings.Split(res, "\n")
	for _, line := range allLines {
		line = strings.TrimSpace(line)
		if line == "" || re.MatchString(line) {
			continue
		}
		lines = append(lines, line)
	}
	root := strings.Join(lines, "\n")
	for _, dep := range deps {
		prefix := strings.TrimSpace(dep.CommandPrefix)
		if prefix[len(prefix)-1:] != "/" {
			prefix += "/"
		}
		command := strings.Replace(strings.TrimSpace(dep.Name), " ", "-", -1) + ".sh"
		fullCommand := prefix + "sds-" + command
		line := strings.TrimSpace(dep.Cron) + " " + strings.TrimSpace(dep.CronEnv) + " " + fullCommand
		root += "\n" + line
	}
	fmt.Printf("%s\n", root)
	return
}

func deployCrontab(ctx *lib.Ctx, deployEnv string) (err error) {
	data, err := ioutil.ReadFile(deploymentsYAML)
	if err != nil {
		fmt.Printf("Error reading file: %s: %+v\n", deploymentsYAML, err)
		return
	}
	var dep sdsDeployment
	err = yaml.Unmarshal(data, &dep)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s: %+v\n", deploymentsYAML, err)
		return
	}
	deployments := []deployment{}
	for _, env := range dep.Environments {
		if env.Name != deployEnv {
			continue
		}
		m := make(map[string]struct{})
		for _, d := range env.Deployments {
			_, ok := m[d.Name]
			if ok {
				return fmt.Errorf("non-unique name %s in %s deploy env", d.Name, deployEnv)
			}
			m[d.Name] = struct{}{}
			deployments = append(deployments, d)
		}
	}
	if len(deployments) == 0 {
		return fmt.Errorf("nothing to deploy for %s env", deployEnv)
	}
	return crontabDeployments(ctx, deployments)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Required deployment environment name: [prod|test]\n")
		os.Exit(1)
	}
	var ctx lib.Ctx
	ctx.TestMode = true
	ctx.Init()
	err := deployCrontab(&ctx, os.Args[1])
	if err != nil {
		fmt.Printf("deployCrontab: %+v\n", err)
	}
}
