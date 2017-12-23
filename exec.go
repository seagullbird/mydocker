package main

import (
	log "github.com/Sirupsen/logrus"
	"strings"
	"os/exec"
	"os"
	_ "github.com/seagullbird/mydocker/nsenter"
)

const ENV_EXEC_PID = "mydocker_pid"
const ENV_EXEC_CMD = "mydocker_cmd"

func ExecContainer(containerName string, cmdArray []string) {
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		log.Errorf("Exec container getContainerPidByName %s error: %v", containerName, err)
		return
	}

	cmdStr := strings.Join(cmdArray, " ")
	log.Infof("container pid %s", pid)
	log.Infof("command %s", cmdStr)

	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	os.Setenv(ENV_EXEC_PID, pid)
	os.Setenv(ENV_EXEC_CMD, cmdStr)

	if err := cmd.Run(); err != nil {
		log.Errorf("Exec container %s error: %v", containerName, err)
	}
}

func getContainerPidByName(containerName string) (string, error) {
	containerInfo, err := GetContainerInfoByName(containerName)
	if err != nil {
		log.Errorf("GetContainerInfoByName with name %s error: %v", containerName, err)
		return "", err
	}
	return containerInfo.Pid, nil
}


