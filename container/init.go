package container

import (
	"os"
	"syscall"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"strings"
	"fmt"
	"os/exec"
	"path/filepath"
)

func RunContainerInitProcess() error {
	cmdArray := readUserCommand()
	if cmdArray == nil || len(cmdArray) == 0 {
		return fmt.Errorf("Run container get user command error, cmdArray is nil")
	}

	setUpMount()
	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		log.Errorf("Exec look path error %v", err)
		return err
	}
	log.Infof("Find path %s", path)
	if err := syscall.Exec(path, cmdArray, os.Environ()); err != nil {
		log.Errorf(err.Error())
	}
	return nil
}

func readUserCommand() []string {
	pipe := os.NewFile(uintptr(3), "pipe")
	msg, err := ioutil.ReadAll(pipe)
	if err != nil {
		log.Errorf("init read pipe error: %v", err)
		return nil
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")
}

func setUpMount() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Errorf("Get current location error: %v", err)
	}
	log.Infof("Current location is %s", pwd)
	pivotRoot(pwd)
	//mount proc
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	// mount dev
	syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID | syscall.MS_STRICTATIME, "mode=755")
}

func pivotRoot(newRoot string) error {
	if err := syscall.Mount(newRoot, newRoot, "bind", syscall.MS_BIND | syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("Mount rootfs to itself error: %v", err)
	}
	// Create <newRoot>/pivot.old directory to put old root
	pivotOld := filepath.Join(newRoot, "pivot.old")
	if err := os.Mkdir(pivotOld, 0777); err != nil {
		return err
	}
	// pivot to new root, old root now mounted on <newRoot>/pivot.old
	if err := syscall.PivotRoot(newRoot, pivotOld); err != nil {
		return fmt.Errorf("pivot_root error: %v", err)
	}
	// Change current directory to new root
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("chdir / error: %v", err)
	}
	pivotOld = filepath.Join("/", "pivot.old")
	//umount pivot.old
	if err := syscall.Unmount(pivotOld, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("umount pivot.old error: %v", err)
	}
	// Delete temp dir
	return os.Remove(pivotOld)
}