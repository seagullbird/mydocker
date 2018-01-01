package main

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/seagullbird/mydocker/cgroups"
	"github.com/seagullbird/mydocker/cgroups/subsystems"
	"github.com/seagullbird/mydocker/container"
	"github.com/seagullbird/mydocker/network"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func Run(tty bool, cmdArray []string, res *subsystems.ResourceConfig, volume, containerName, imageName string, envSlice []string, nw string, portmapping []string) {
	containerID := randStringBytes(10)
	if containerName == "" {
		containerName = containerID
	}
	parent, writePipe := container.NewParentProcess(tty, volume, containerName, imageName, envSlice)
	if parent == nil {
		log.Errorf("New parent process error")
		return
	}

	if err := parent.Start(); err != nil {
		log.Error(err)
	}

	cgroupManager := cgroups.NewCgroupManager("mydocker")
	defer cgroupManager.Destroy()
	// Set resources limitation
	cgroupManager.Set(res)
	// Add container process into each cgroup
	cgroupManager.Apply(parent.Process.Pid, res)

	containerInfo := &container.ContainerInfo{
		Id:   containerID,
		Pid:  strconv.Itoa(parent.Process.Pid),
		Name: containerName,
	}

	if nw != "" {
		// config container network
		network.Init()
		containerInfo.Network = nw
		containerInfo.PortMapping = portmapping
		ip, err := network.Connect(nw, containerInfo)
		if err != nil {
			log.Errorf("Error Connect Network %v", err)
			return
		}
		containerInfo.IPAddress = ip
	}

	if err := recordContainerInfo(containerInfo, cmdArray, volume, imageName); err != nil {
		log.Errorf("Record container info error: %v", err)
		return
	}

	// initialize the container
	sendInitCommand(cmdArray, writePipe)
	if tty {
		parent.Wait()
		if nw != "" {
			network.Disconnect(nw, containerInfo)
		}
		container.DeleteWorkSpace(volume, containerName, imageName)
		deleteContainerInfo(containerName)
	}
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

func recordContainerInfo(containerInfo *container.ContainerInfo, cmdArray []string, volume, imageName string) error {
	createdTime := time.Now().Format("2006-01-01 15:00:00")
	command := strings.Join(cmdArray, "")

	containerInfo.Command = command
	containerInfo.CreatedTime = createdTime
	containerInfo.Status = container.RUNNING
	containerInfo.Volume = volume
	containerInfo.Image = imageName

	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("Record container info error %v", err)
		return err
	}
	jsonStr := string(jsonBytes)
	containerInfoDir := fmt.Sprintf(container.DefaultInfoLocation, containerInfo.Name)
	if err := os.MkdirAll(containerInfoDir, 0622); err != nil {
		log.Errorf("Mkdir error %s error: %v", containerInfoDir, err)
		return err
	}
	fileName := filepath.Join(containerInfoDir, container.ConfigName)
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		log.Errorf("Create file %s error: %v", fileName, err)
		return err
	}
	if _, err := file.WriteString(jsonStr); err != nil {
		log.Errorf("File write string error: %v", err)
		return err
	}
	return nil
}

func deleteContainerInfo(containerName string) {
	containerInfoDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.RemoveAll(containerInfoDir); err != nil {
		log.Errorf("Remove dir %s error %v", containerInfoDir, err)
	}
}
