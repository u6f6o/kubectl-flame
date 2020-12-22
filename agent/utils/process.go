package utils

import (
	"errors"
	"github.com/VerizonMedia/kubectl-flame/agent/details"
	"github.com/VerizonMedia/kubectl-flame/api"
	"github.com/fntlnz/mountinfo"
	"io"
	"os"
	"path"
	"strings"
)

var (
	defaultProcessNames = map[api.ProgrammingLanguage]string{
		api.Java: "java",
	}
)

func getProcessName(job *details.ProfilingJob) string {
	if job.TargetProcessName != "" {
		return job.TargetProcessName
	}

	if val, ok := defaultProcessNames[job.Language]; ok {
		return val
	}

	return ""
}

func FindProcessId(job *details.ProfilingJob) (string, error) {
	name := getProcessName(job)

	api.PublishError(errors.New("Process name: " + name))

	foundProc := ""
	proc, err := os.Open("/proc")
	if err != nil {
		return "", err
	}

	defer proc.Close()

	for {
		dirs, err := proc.Readdir(15)

		api.PublishError(errors.New("Process name: " + name))

		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		for _, di := range dirs {

			api.PublishError(errors.New("Trying dir: " + di.Name()))

			if !di.IsDir() {
				continue
			}

			dname := di.Name()
			if dname[0] < '0' || dname[0] > '9' {
				api.PublishError(errors.New("Dir name too short/long, skipping:" + di.Name()))
				continue
			}

			mi, err := mountinfo.GetMountInfo(path.Join("/proc", dname, "mountinfo"))
			if err != nil {
				api.PublishError(errors.New("Can't get mount info for " + di.Name()))
				continue
			}

			for _, m := range mi {
				root := m.Root
				if strings.Contains(root, job.PodUID) &&
					strings.Contains(root, job.ContainerName) {

					exeName, err := os.Readlink(path.Join("/proc", dname, "exe"))
					if err != nil {
						api.PublishError(errors.New("Can't read proc link for " + m.Root + " and " + dname))
						continue
					}

					if name != "" {
						// search by process name
						if strings.Contains(exeName, name) {
							return dname, nil
						}
					} else {
						if foundProc != "" {
							return "", errors.New("found more than one process on container," +
								" specify process name using --pgrep flag")
						} else {
							foundProc = dname
						}
					}
				}
			}
		}
	}

	if foundProc != "" {
		return foundProc, nil
	}

	return "", errors.New("could not find any process")
}
