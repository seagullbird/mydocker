package container

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//Create an Overlay filesystem as container root workspace
func NewWorkSpace(volume, imageName, containerName string) {
	CreateReadOnlyLayer(imageName)
	CreateWriteLayer(containerName)
	// For overlayFS
	CreateWorkdir(containerName)
	CreateMountPoint(containerName, imageName)
	if volume != "" {
		volumeDirs := volumeDirExtract(volume)
		length := len(volumeDirs)
		if length == 2 && volumeDirs[0] != "" && volumeDirs[1] != "" {
			MountVolume(volumeDirs, containerName)
			log.Infof("%q", volumeDirs)
		} else {
			log.Infof("Volume parameter input is not correct. %s", volume)
		}
	}
}

func CreateReadOnlyLayer(imageName string) {
	imageDir := layerPath(imageName)
	exist, err := PathExists(imageDir)
	if err != nil {
		log.Infof("Fail to judge whether dir %s exists. %v", imageDir, err)
		return
	}
	if exist == false {
		log.Errorf("Image not found. Run docker pull <image> && docker export first.")
	}
}

func CreateWriteLayer(containerName string) {
	writeDir := containerWriteLayerPath(containerName)
	if err := os.MkdirAll(writeDir, 0777); err != nil {
		log.Errorf("Mkdir dir %s error. %v", writeDir, err)
	}
}

func CreateWorkdir(containerName string) {
	workDir := containerWorkPath(containerName, "image")
	if err := os.MkdirAll(workDir, 0777); err != nil {
		log.Errorf("Mkdir dir %s error. %v", workDir, err)
	}
}

func CreateMountPoint(containerName, imageName string) {
	mntDir := ContainerMntPath(containerName)
	if err := os.MkdirAll(mntDir, 0777); err != nil {
		log.Errorf("Mkdir dir %s error. %v", mntDir, err)
	}
	dirs := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
		layerPath(imageName),
		containerWriteLayerPath(containerName),
		containerWorkPath(containerName, "image"))
	cmd := exec.Command("mount", "-t", "overlay", "-o", dirs, "none", mntDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("%v", err)
	}
}

func DeleteWorkSpace(volume, containerName, imageName string) {
	if volume != "" {
		volumeDirs := volumeDirExtract(volume)
		length := len(volumeDirs)
		if length == 2 && volumeDirs[0] != "" && volumeDirs[1] != "" {
			UmountVolume(volumeDirs, containerName)
		}
	}
	UnmountMountPoint(containerName)
	containerLayerPath := layerPath(containerName)
	if err := os.RemoveAll(containerLayerPath); err != nil {
		log.Errorf("Remove dir %s error: %v", containerLayerPath, err)
	}
}

func UnmountMountPoint(containerName string) {
	mntDir := ContainerMntPath(containerName)
	cmd := exec.Command("umount", mntDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Umount mntDir error: %v", err)
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

func volumeDirExtract(volume string) []string {
	var volumeDirs []string
	volumeDirs = strings.Split(volume, ":")
	return volumeDirs
}

func MountVolume(volumeDirs []string, containerName string) {
	volumeLowerDir := layerPath("volume_lowerdir")
	volumeUpperDir := volumeDirs[0]
	volumeWorkDir := containerWorkPath(containerName, "volume")

	if err := os.Mkdir(volumeLowerDir, 0777); err != nil {
		log.Infof("Mkdir host dir %s error: %v", volumeLowerDir, err)
	}
	if err := os.Mkdir(volumeWorkDir, 0777); err != nil {
		log.Infof("Mkdir host dir %s error: %v", volumeWorkDir, err)
	}
	// Create mount point inside container
	containerDir := volumeDirs[1]
	containerVolumeDir := filepath.Join(ContainerMntPath(containerName), containerDir)
	if err := os.Mkdir(containerVolumeDir, 0777); err != nil {
		log.Infof("Mkdir container dir %s error: %v", containerVolumeDir, err)
	}
	dirs := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
		volumeLowerDir,
		volumeUpperDir,
		volumeWorkDir)
	cmd := exec.Command("mount", "-t", "overlay", "-o", dirs, "none", containerVolumeDir)
	if err := cmd.Run(); err != nil {
		log.Errorf("Mount volume failed error: %v", err)
	}
}

func UmountVolume(volumeDirs []string, containerName string) {
	containerDir := volumeDirs[1]
	containerVolumeDir := filepath.Join(ContainerMntPath(containerName), containerDir)
	cmd := exec.Command("umount", containerVolumeDir)
	if err := cmd.Run(); err != nil {
		log.Errorf("Umount volume failed error: %v", err)
	}
}
