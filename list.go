package main

import (
	"os"
	"github.com/seagullbird/mydocker/container"
	"fmt"
	"strings"
	"io/ioutil"
	log "github.com/Sirupsen/logrus"
	"path/filepath"
	"encoding/json"
	"text/tabwriter"
)

func ListContainers() {
	containersInfoDir := strings.TrimSuffix(fmt.Sprintf(container.DefaultInfoLocation, ""), "/")
	files, err := ioutil.ReadDir(containersInfoDir)
	if err != nil {
		log.Errorf("Read dir %s error: %v", containersInfoDir, err)
		return
	}
	var containerInfos []*container.ContainerInfo
	for _, file := range files {
		containerInfo, err := getContainerInfo(file)
		if err != nil {
			log.Errorf("Get container %s info error: %v", file.Name(), err)
			continue
		}
		containerInfos = append(containerInfos, containerInfo)
	}

	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "ID\tNAME\tPID\tIMAGE\tSTATUS\tCOMMAND\tCREATED\n")
	for _, item := range containerInfos {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Id,
			item.Name,
			item.Pid,
			item.Image,
			item.Status,
			item.Command,
			item.CreatedTime)
	}
	if err := w.Flush(); err != nil {
		log.Errorf("Flush error: %v", err)
		return
	}
}

func getContainerInfo(file os.FileInfo) (*container.ContainerInfo, error) {
	containerName := file.Name()
	configFileDir := filepath.Join(fmt.Sprintf(container.DefaultInfoLocation, containerName), container.ConfigName)
	content, err := ioutil.ReadFile(configFileDir)
	if err != nil {
		log.Errorf("Read file %s error: %v", configFileDir, err)
		return nil, err
	}
	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(content, &containerInfo); err != nil {
		log.Errorf("Json unmarshal error: %v", err)
		return nil, err
	}
	return &containerInfo, nil
}
