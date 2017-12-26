package subsystems

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

type CpusetSubSystem struct {
}

func (s *CpusetSubSystem) Name() string {
	return "cpuset"
}

func (s *CpusetSubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, true); err == nil {
		if res.CpuSet != "" {
			log.Infof("Setting cpuset limit: %v", res.CpuSet)
			// cpuset.mems also need to be configured
			// https://www.richardhsu.me/posts/2014/12/08/cgroups-and-no-space.html
			// https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/6/html/resource_management_guide/sec-cpuset
			if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "cpuset.mems"), []byte("0"), 0644); err != nil {
				return fmt.Errorf("set cgroup cpuset.mems fail %v", err)
			}
			// write cpuset limit into cpuset.cpus
			if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "cpuset.cpus"), []byte(res.CpuSet), 0644); err != nil {
				return fmt.Errorf("set cgroup cpuset.cpus fail %v", err)
			}
		}
		return nil
	} else {
		return err
	}
}

func (s *CpusetSubSystem) Apply(cgroupPath string, pid int, res *ResourceConfig) error {
	if res.CpuSet == "" {
		return nil
	}
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {
		// write process pid into tasks
		if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), 0644); err != nil {
			return fmt.Errorf("set cgroup proc fail %v", err)
		}
		return nil
	} else {
		return fmt.Errorf("get cgroup %s error: %v", cgroupPath, err)
	}
}

func (s *CpusetSubSystem) Remove(cgroupPath string) error {
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {
		return os.Remove(subsysCgroupPath)
	} else {
		return err
	}
}
