package main


import (
	log "github.com/Sirupsen/logrus"
	"fmt"
	"os/exec"
)

func commitContainer(imageName string){
	mntURL := "/root/mydocker_images/mnt"
	imageTar := "/root/mydocker_images/" + imageName + ".tar"
	fmt.Printf("%s",imageTar)
	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntURL, ".").CombinedOutput(); err != nil {
		log.Errorf("Tar folder %s error %v", mntURL, err)
	}
}