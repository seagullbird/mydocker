package container

import (
	"syscall"
	"os/exec"
	"os"
	log "github.com/Sirupsen/logrus"
)

func NewParentProcess (tty bool) (*exec.Cmd, *os.File) {
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
	}
	homeDir := "/root/mydocker_images/"
	mntDir := homeDir + "mnt/"
	cmd.Dir = mntDir
	NewWorkSpace(homeDir, mntDir)
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
func NewWorkSpace(homeDir string, mntDir string) {
	lowerDir := CreateReadOnlyLayer(homeDir)
	upperDir := CreateWriteLayer(homeDir)
	// For overlayFS
	workDir := CreateWorkdir(homeDir)
	CreateMountPoint(lowerDir, upperDir, workDir, mntDir)
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

func DeleteWorkSpace(homeDir string, mntDir string) {
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
