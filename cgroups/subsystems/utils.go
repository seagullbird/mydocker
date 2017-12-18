package subsystems

import (
	"os"
	"bufio"
	"strings"
	"path"
	"fmt"
)

func FindCgroupMountPoint(subsystem string) string {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		fields := strings.Split(txt, " ")
		for _, opt := range strings.Split(fields[len(fields)-1], ",") {
			if opt == subsystem {
				return fields[4]
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return ""
	}

	return ""
}

func GetCgroupPath(subsystem string, cgroupPath string, autoCreate bool) (string, error) {
	cgroupRoot := FindCgroupMountPoint(subsystem)
	cgroupAbsPath := path.Join(cgroupRoot, cgroupPath)
	if _, err := os.Stat(cgroupAbsPath); err == nil || (autoCreate && os.IsNotExist(err)) {
		if os.IsNotExist(err) {
			if err := os.Mkdir(cgroupAbsPath, 0755); err != nil {
			} else {
				return "", fmt.Errorf("error creating cgroup %v", err)
			}
		}
		return cgroupAbsPath, nil
	} else {
		return "", fmt.Errorf("cgroup path error %v", err)
	}
}