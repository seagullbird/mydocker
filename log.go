package main

import (
	"fmt"
	"github.com/seagullbird/mydocker/container"
	"path/filepath"
	"os"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
)

func logContainer(containerName string) {
	containerInfoDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	logFileLocation := filepath.Join(containerInfoDir, container.ContainerLogFile)
	file, err := os.Open(logFileLocation)
	defer file.Close()
	if err != nil {
		log.Errorf("Open log file %s error: %v", logFileLocation, err)
		return
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Errorf("Read log file %s error: %v", logFileLocation, err)
		return
	}
	fmt.Fprint(os.Stdout, string(content))
}
