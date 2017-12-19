package main

import (
	"github.com/seagullbird/mydocker/container"
	log "github.com/Sirupsen/logrus"
	"os"
	"github.com/seagullbird/mydocker/cgroups/subsystems"
	"strings"
	"github.com/seagullbird/mydocker/cgroups"
)

func Run(tty bool, cmdArray []string, res *subsystems.ResourceConfig) {
	parent, writePipe := container.NewParentProcess(tty)
	if parent == nil {
		log.Errorf("New parent process error")
		return
	}

	if err := parent.Start(); err != nil {
		log.Error(err)
	}
	cgroupManager := cgroups.NewCgroupManager("mydocker-cgroup")
	defer cgroupManager.Destroy()
	// Set resources limitation
	cgroupManager.Set(res)
	// Add container process into each cgroup
	cgroupManager.Apply(parent.Process.Pid)
	// initialize the container
	sendInitCommand(cmdArray, writePipe)
	parent.Wait()
	os.Exit(0)
}

func sendInitCommand(cmdArray []string, writePipe *os.File) {
	command := strings.Join(cmdArray, " ")
	log.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}
