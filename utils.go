package main

import (
	"encoding/json"
	"fmt"
	"github.com/seagullbird/mydocker/container"
	"io/ioutil"
	"path/filepath"
)

func GetContainerInfoByName(containerName string) (*container.ContainerInfo, error) {
	containerInfoDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configPath := filepath.Join(containerInfoDir, container.ConfigName)
	contentBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		return nil, err
	}
	return &containerInfo, nil
}
