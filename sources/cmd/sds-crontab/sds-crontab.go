package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

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
	TempDir       string            `yaml:"temp_dir"`
	Env           map[string]string `yaml:"env"`
	Disabled      bool              `yaml:"disabled"`
}

type environment struct {
	Name        string       `yaml:"name"`
	Deployments []deployment `yaml:"deployments"`
}

type sdsDeployment struct {
	Environments []environment `yaml:"environments"`
}

func createCommand(deployEnv string, dep deployment, cmd string) (err error) {
	// SDS_CSV_PREFIX='/root/jobs' SDS_SILENT=1 SDS_TASK_TIMEOUT_SECONDS=43200 SDS_NCPUS_SCALE=2 SDS_SCROLL_SIZE=500 SDS_ES_BULKSIZE=500 SDS_TASKS_SKIP_RE='^sds-cncf-' /usr/bin/sds-cron-task.sh sds_main prod 1>> /tmp/sds_main.log 2>>/tmp/sds_main.err
	// SDS_CSV_PREFIX='/root/cncf_jobs' SDS_NCPUS_SCALE=0.3 SDS_SILENT=1 SDS_TASK_TIMEOUT_SECONDS=43200 SDS_SCROLL_SIZE=1000 SDS_ES_BULKSIZE=1000 SDS_ONLY_P2O=1 SDS_TASKS_RE='^sds-cncf-' /usr/bin/sds-cron-task.sh sds_cncf prod 1>> /tmp/sds_cncf.log 2>>/tmp/sds_cncf.err
	root := ""
	if !dep.Master {
		root += "SDS_ONLY_P2O=1 "
	}
	var (
		pref   string
		prefix string
		ok     bool
	)
	pref, ok = dep.Env["SDS_CSV_PREFIX"]
	if ok {
		prefix = pref
		delete(dep.Env, "SDS_CSV_PREFIX")
	} else {
		prefix = dep.CSVPrefix
	}
	prefix = strings.TrimSpace(prefix)
	if prefix[len(prefix)-1:] != "/" {
		prefix += "/"
	}
	name := "sds-" + strings.Replace(strings.TrimSpace(dep.Name), " ", "-", -1)
	root += "SDS_CSV_PREFIX=" + prefix + name + " "
	//  /usr/bin/sds-cron-task.sh sds_main prod 1>> /tmp/sds_main.log 2>>/tmp/sds_main.err
	//  /usr/bin/sds-cron-task.sh sds_cncf prod 1>> /tmp/sds_cncf.log 2>>/tmp/sds_cncf.err
	for k, v := range dep.Env {
		root += k + "='" + v + "' "
	}
	prefix = strings.TrimSpace(dep.CommandPrefix)
	if prefix[len(prefix)-1:] != "/" {
		prefix += "/"
	}
	tmp := strings.TrimSpace(dep.TempDir)
	if tmp[len(tmp)-1:] != "/" {
		tmp += "/"
	}
	tmpName := tmp + name
	root += fmt.Sprintf("%ssds-cron-task.sh %s %s 1>> %s.log 2>>%s.err", prefix, name, deployEnv, tmpName, tmpName)
	err = ioutil.WriteFile(cmd, []byte("#/bin/bash\n"+root+"\n"), 0755)
	if err != nil {
		return
	}
	fmt.Printf("Processed file: %s: %s\n", cmd, root)
	return
}

func crontabDeployments(ctx *lib.Ctx, deployEnv string, deps []deployment) (err error) {
	reS := "("
	for _, d := range deps {
		name := strings.Replace(strings.TrimSpace(d.Name), " ", "-", -1)
		reS += "sds-" + name + "|"
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
		// */10 * * * * PATH=$PATH:/snap/bin /usr/bin/sds-main.sh
		prefix := strings.TrimSpace(dep.CommandPrefix)
		if prefix[len(prefix)-1:] != "/" {
			prefix += "/"
		}
		command := strings.Replace(strings.TrimSpace(dep.Name), " ", "-", -1) + ".sh"
		fullCommand := prefix + "sds-" + command
		line := strings.TrimSpace(dep.Cron) + " " + strings.TrimSpace(dep.CronEnv) + " " + fullCommand
		if dep.Disabled {
			line = "# " + line
		}
		fmt.Printf("crontab entry: %s\n", line)
		root += "\n" + line
		err = createCommand(deployEnv, dep, fullCommand)
		if err != nil {
			return
		}
	}
	root += "\n"
	var tmp *os.File
	tmp, err = ioutil.TempFile("", "")
	if err != nil {
		return
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	err = ioutil.WriteFile(tmp.Name(), []byte(root), 0644)
	if err != nil {
		return
	}
	res, err = lib.ExecCommand(ctx, []string{"crontab", tmp.Name()}, nil, nil)
	if err != nil {
		return
	}
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
		hasMaster := false
		for _, d := range env.Deployments {
			if d.Disabled {
				lib.Printf("%s will be added as disabled\n", d.Name)
				continue
			}
			_, ok := m[d.Name]
			if ok {
				return fmt.Errorf("non-unique name %s in %s deploy env", d.Name, deployEnv)
			}
			if d.Master {
				if !hasMaster {
					hasMaster = true
				} else {
					return fmt.Errorf("there must be exactly one master deployment in %s deploy env, found multiple", deployEnv)
				}
			}
			m[d.Name] = struct{}{}
			deployments = append(deployments, d)
		}
		if !hasMaster {
			return fmt.Errorf("there must be exactly one master deployment in %s deploy env, found none", deployEnv)
		}
	}
	if len(deployments) == 0 {
		return fmt.Errorf("nothing to deploy for %s env", deployEnv)
	}
	return crontabDeployments(ctx, deployEnv, deployments)
}

func main() {
	fmt.Printf("sds-crontab: %+v: start\n", time.Now())
	if len(os.Args) < 2 {
		fmt.Printf("Required deployment environment name: [prod|test]\n")
		os.Exit(1)
	}
	var ctx lib.Ctx
	ctx.TestMode = true
	ctx.Init()
	_ = os.Setenv("SDS_SIMPLE_PRINTF", "1")
	err := deployCrontab(&ctx, os.Args[1])
	if err != nil {
		fmt.Printf("deployCrontab: %+v\n", err)
		return
	}
	fmt.Printf("sds-crontab: %+v: all OK\n", time.Now())
}
