package main

import (
	"github.com/seagullbird/mydocker/container"
	log "github.com/Sirupsen/logrus"
	"os"
	"github.com/seagullbird/mydocker/cgroups/subsystems"
	"strings"
	"github.com/seagullbird/mydocker/cgroups"
	"math/rand"
	"time"
	"strconv"
	"encoding/json"
	"fmt"
	"path/filepath"
)

func Run(tty bool, cmdArray []string, res *subsystems.ResourceConfig, volume string, containerName string) {
	parent, writePipe := container.NewParentProcess(tty, volume)
	if parent == nil {
		log.Errorf("New parent process error")
		return
	}

	if err := parent.Start(); err != nil {
		log.Error(err)
	}
	containerName, err := recordContainerInfo(parent.Process.Pid, cmdArray, containerName)
	if err != nil {
		log.Errorf("Record container info error: %v", err)
		return
	}

	cgroupManager := cgroups.NewCgroupManager("mydocker-cgroup")
	defer cgroupManager.Destroy()
	// Set resources limitation
	cgroupManager.Set(res)
	// Add container process into each cgroup
	cgroupManager.Apply(parent.Process.Pid)
	// initialize the container
	sendInitCommand(cmdArray, writePipe)
	if tty {
		parent.Wait()
		homeDir := "/root/mydocker_images/"
		mntDir := homeDir + "mnt/"
		container.DeleteWorkSpace(homeDir, mntDir, volume)
	}
	os.Exit(0)
}

func sendInitCommand(cmdArray []string, writePipe *os.File) {
	command := strings.Join(cmdArray, " ")
	log.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}

func randStringBytes(n int) string {
	letterBytes := "1234567890"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func recordContainerInfo(containerPID int, cmdArray []string, containerName string) (string, error) {
	id := randStringBytes(10)
	createdTime := time.Now().Format("2006-01-01 15:00:00")
	command := strings.Join(cmdArray, "")
	if containerName == "" {
		containerName = id
	}
	containerInfo := &container.ContainerInfo{
		Id: 			id,
		Pid: 			strconv.Itoa(containerPID),
		Command: 		command,
		CreatedTime:	createdTime,
		Status: 		container.RUNNING,
		Name: 			containerName,
	}
	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("Record container info error %v", err)
		return "", err
	}
	jsonStr := string(jsonBytes)
	containerInfoDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.MkdirAll(containerInfoDir, 0622); err != nil {
		log.Errorf("Mkdir error %s error: %v", containerInfoDir, err)
		return "", err
	}
	fileName := filepath.Join(containerInfoDir, container.ConfigName)
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		log.Errorf("Create file %s error: %v", fileName, err)
		return "", err
	}
	if _, err := file.WriteString(jsonStr); err != nil {
		log.Errorf("File write string error: %v", err)
		return "", err
	}
	return containerName, nil
}

func deleteContainerInfo(containerId string) {
	containerInfoDir := fmt.Sprintf(container.DefaultInfoLocation, containerId)
	if err := os.RemoveAll(containerInfoDir); err != nil {
		log.Errorf("Remove dir %s error %v", containerInfoDir, err)
	}
}