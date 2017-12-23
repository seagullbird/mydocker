package container

import (
	"syscall"
	"os/exec"
	"os"
	log "github.com/Sirupsen/logrus"
	"strings"
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
}

var (
	RUNNING				string = "running"
	STOP 				string = "stopped"
	Exit 				string = "exited"
	DefaultInfoLocation string = "/var/run/mydocker/%s/"
	ConfigName			string = "config.json"
	ContainerLogFile	string = "container.log"
)

func NewParentProcess (tty bool, volume string, containerName string) (*exec.Cmd, *os.File) {
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
	homeDir := "/root/mydocker_images/"
	mntDir := homeDir + "mnt/"
	cmd.Dir = mntDir
	NewWorkSpace(homeDir, mntDir, volume)
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

//Create an Overlay filesystem as container root workspace
func NewWorkSpace(homeDir string, mntDir string, volume string) {
	lowerDir := CreateReadOnlyLayer(homeDir)
	upperDir := CreateWriteLayer(homeDir)
	// For overlayFS
	workDir := CreateWorkdir(homeDir)
	CreateMountPoint(lowerDir, upperDir, workDir, mntDir)
	if volume != "" {
		volumeDirs := volumeDirExtract(volume)
		length := len(volumeDirs)
		if length == 2 && volumeDirs[0] != "" && volumeDirs[1] != "" {
			MountVolume(homeDir, mntDir, volumeDirs)
			log.Infof("%q", volumeDirs)
		} else {
			log.Infof("Volume parameter input is not correct.")
		}
	}
}

func CreateReadOnlyLayer(homeDir string) string {
	// here busyboxDir is the ReadOnlyLayer
	busyboxDir := homeDir + "busybox/"
	busyboxTarDir := homeDir + "busybox.tar"
	exist, err := PathExists(busyboxDir)
	if err != nil {
		log.Infof("Fail to judge whether dir %s exists. %v", busyboxDir, err)
	}
	if exist == false {
		if err := os.Mkdir(busyboxDir, 0777); err != nil {
			log.Errorf("Mkdir dir %s error. %v", busyboxDir, err)
		}
		if _, err := exec.Command("tar", "-xvf", busyboxTarDir, "-C", busyboxDir).CombinedOutput(); err != nil {
			log.Errorf("Untar dir %s error %v", busyboxDir, err)
		}
	}
	return busyboxDir
}

func CreateWriteLayer(homeDir string) string {
	writeDir := homeDir + "writeLayer/"
	if err := os.Mkdir(writeDir, 0777); err != nil {
		log.Errorf("Mkdir dir %s error. %v", writeDir, err)
	}
	return writeDir
}

func CreateWorkdir(homeDir string) string {
	workDir := homeDir + "workDir/"
	if err := os.Mkdir(workDir, 0777); err != nil {
		log.Errorf("Mkdir dir %s error. %v", workDir, err)
	}
	return workDir
}

func CreateMountPoint(lowerDir string, upperDir string, workDir string, mntDir string) {
	if err := os.Mkdir(mntDir, 0777); err != nil {
		log.Errorf("Mkdir dir %s error. %v", mntDir, err)
	}
	dirs := "lowerdir=" + lowerDir + "," + "upperdir=" + upperDir + "," + "workdir=" + workDir
	cmd := exec.Command("mount", "-t", "overlay", "-o", dirs, "none", mntDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("%v", err)
	}
}

func DeleteWorkSpace(homeDir string, mntDir string, volume string) {
	if volume != "" {
		volumeDirs := volumeDirExtract(volume)
		length := len(volumeDirs)
		if length == 2 && volumeDirs[0] != "" && volumeDirs[1] != "" {
			UmountVolume(homeDir, mntDir, volumeDirs)
		}
	}
	DeleteMountPoint(homeDir, mntDir)
	DeleteWriteLayer(homeDir)
	DeleteWorkDir(homeDir)
}

func DeleteMountPoint(homeDir string, mntDir string) {
	cmd := exec.Command("umount", mntDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Umount mntDir error: %v", err)
	}
	if err := os.RemoveAll(mntDir); err != nil {
		log.Errorf("Remove dir %s error: %v", mntDir, err)
	}
}

func DeleteWriteLayer(homeDir string) {
	writeDir := homeDir + "writeLayer"
	if err := os.RemoveAll(writeDir); err != nil {
		log.Errorf("Remove dir %s error: %v", writeDir, err)
	}
}

func DeleteWorkDir(homeDir string) {
	workDir := homeDir + "workDir"
	if err := os.RemoveAll(workDir); err != nil {
		log.Errorf("Remove dir %s error: %v", workDir, err)
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func volumeDirExtract(volume string) ([]string) {
	var volumeDirs []string
	volumeDirs = strings.Split(volume, ":")
	return volumeDirs
}

func MountVolume(homeDir string, mntDir string, volumeDirs []string) {
	// Create host directory
	hostDir := volumeDirs[0]
	volumeLowerDir := filepath.Join(hostDir, "lowerdir")
	volumeUpperDir := filepath.Join(hostDir, "volume")
	volumeWorkDir := filepath.Join(hostDir, "workdir")
	if err := os.Mkdir(hostDir, 0777); err != nil {
		log.Infof("Mkdir host dir %s error: %v", hostDir, err)
	}
	if err := os.Mkdir(volumeLowerDir, 0777); err != nil {
		log.Infof("Mkdir host dir %s error: %v", volumeLowerDir, err)
	}
	if err := os.Mkdir(volumeUpperDir, 0777); err != nil {
		log.Infof("Mkdir host dir %s error: %v", volumeUpperDir, err)
	}
	if err := os.Mkdir(volumeWorkDir, 0777); err != nil {
		log.Infof("Mkdir host dir %s error: %v", volumeWorkDir, err)
	}
	// Create mount point inside container
	containerDir := volumeDirs[1]
	containerVolumeDir := mntDir + strings.Trim(containerDir, "/")
	if err := os.Mkdir(containerVolumeDir, 0777); err != nil {
		log.Infof("Mkdir container dir %s error: %v", containerVolumeDir, err)
	}
	dirs := "lowerdir=" + volumeLowerDir + "," + "upperdir=" + volumeUpperDir + "," + "workdir=" + volumeWorkDir
	cmd := exec.Command("mount", "-t", "overlay", "-o", dirs, "none", containerVolumeDir)
	if err := cmd.Run(); err != nil {
		log.Errorf("Mount volume failed error: %v", err)
	}
}

func UmountVolume(homeDir string, mntDir string, volumeDirs []string) {
	containerDir := volumeDirs[1]
	containerVolumeDir := mntDir + containerDir
	cmd := exec.Command("umount", containerVolumeDir)
	if err := cmd.Run(); err != nil {
		log.Errorf("Umount volume failed error: %v", err)
	}
}