package main

import (
	log "github.com/Sirupsen/logrus"
	"syscall"
	"strconv"
	"github.com/seagullbird/mydocker/container"
	"encoding/json"
	"fmt"
	"path/filepath"
	"io/ioutil"
)

func stopContainer(containerName string) {
	containerInfo, err := GetContainerInfoByName(containerName)
	if err != nil {
		log.Errorf("Get container %s info error %v", containerName, err)
		return
	}
	pid, _ := strconv.Atoi(containerInfo.Pid)
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		log.Errorf("Stop container %s error %v", containerName, err)
		return
	}
	containerInfo.Status = container.STOP
	containerInfo.Pid = " "
	newContentBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("Json marshal %s error %v", containerName, err)
		return
	}
	containerInfoDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := filepath.Join(containerInfoDir, container.ConfigName)
	if err := ioutil.WriteFile(configFilePath, newContentBytes, 0622); err != nil {
		log.Errorf("Write file %s error", configFilePath, err)
	}
}
