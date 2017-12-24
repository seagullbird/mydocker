package container

import (
	"syscall"
	"os/exec"
	"os"
	log "github.com/Sirupsen/logrus"
	"path/filepath"
	"fmt"
)

type ContainerInfo struct {
	Pid			string `json:pid`
	Id			string `json:id`
	Name 		string `json:name`
	Command 	string `json:command`
	CreatedTime	string `json:createTime`
	Status 		string `json:status`
	Volume		string `json:volume`
	Image		string `json:image`
}

var (
	RUNNING				string = "running"
	STOP 				string = "stopped"
	Exit 				string = "exited"
	ConfigName			string = "config.json"
	ContainerLogFile	string = "container.log"
	RootDir				string = "/var/lib/mydocker/"
	DefaultInfoLocation string = filepath.Join(RootDir, "containers/%s/")
	LayerDir			string = filepath.Join(RootDir, "overlay2/%s/")
	MntDir				string = filepath.Join(RootDir, "overlay2/%s/merged/")
	WriteLayerDir		string = filepath.Join(RootDir, "overlay2/%s/write_layer/")
	WorkDir				string = filepath.Join(RootDir, "overlay2/%s/work/%s/")
)

func layerPath(imageName string) string {
	return fmt.Sprintf(LayerDir, imageName)
}

func ContainerMntPath(containerName string) string {
	return fmt.Sprintf(MntDir, containerName)
}

func containerWriteLayerPath(containerName string) string {
	return fmt.Sprintf(WriteLayerDir, containerName)
}

func containerWorkPath(containerName, sub string) string {
	return fmt.Sprintf(WorkDir, containerName, sub)
}

func NewParentProcess (tty bool, volume, containerName, imageName string) (*exec.Cmd, *os.File) {
	readPipe, writePipe, err := NewPipe()
	if err != nil {
		log.Errorf("New pipe error %v", err)
		return nil, nil
	}

	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWIPC | syscall.CLONE_NEWNET |
			syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS,
	}

	if tty {
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
	} else {
		// save container stdout
		containerInfoDir := fmt.Sprintf(DefaultInfoLocation, containerName)
		if err := os.MkdirAll(containerInfoDir, 0622); err != nil {
			log.Errorf("NewParentProcess mkdir %s error: %v", containerInfoDir, err)
			return nil, nil
		}
		stdLogFilePath := filepath.Join(containerInfoDir, ContainerLogFile)
		stdLogFile, err := os.Create(stdLogFilePath)
		if err != nil {
			log.Errorf("NewParentProcess create file %s error: %v", stdLogFilePath, err)
			return nil, nil
		}
		cmd.Stdout = stdLogFile
	}

	cmd.Dir = ContainerMntPath(containerName)
	NewWorkSpace(volume, imageName, containerName)
	cmd.ExtraFiles = []*os.File{readPipe}
	return cmd, writePipe
}

func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	} else {
		return read, write, nil
	}
}
