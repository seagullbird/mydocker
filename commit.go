package main


import (
	log "github.com/Sirupsen/logrus"
	"fmt"
	"os/exec"
	"github.com/seagullbird/mydocker/container"
)

func commitContainer(packageName, containerName string){
	mntDir := container.ContainerMntPath(containerName)
	exists, err := container.PathExists(mntDir)
	if err != nil || exists == false {
		log.Errorf("Cannot find container mount point %s error: %v", mntDir, err)
		return
	}
	if packageName == "" {
		packageName = "image"
	}
	imageTar := fmt.Sprintf("/root/%s.tar", packageName)
	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntDir, ".").CombinedOutput(); err != nil {
		log.Errorf("Tar folder %s error %v", mntDir, err)
	}
	fmt.Printf("Exported to %s\n", imageTar)
}
