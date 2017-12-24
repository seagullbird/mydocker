package main

import (
	log "github.com/Sirupsen/logrus"
	"strings"
	"os/exec"
	"os"
	_ "github.com/seagullbird/mydocker/nsenter"
	"fmt"
	"io/ioutil"
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
	// get container envs
	containerEnvs := getEnvsByPid(pid)
	cmd.Env = append(os.Environ(), containerEnvs...)

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

func getEnvsByPid(pid string) []string {
	// /proc/PID/environ saves process's env
	path := fmt.Sprintf("/proc/%s/environ", pid)
	contentBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Errorf("Read file %s error: %v", path, err)
		return nil
	}
	// \u0000 separates multiple envs
	envs := strings.Split(string(contentBytes), "\u0000")
	return envs
}