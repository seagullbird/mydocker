package cgroups

import (
	"github.com/seagullbird/mydocker/cgroups/subsystems"
	log "github.com/Sirupsen/logrus"
)

type CgroupManager struct {
	Path string
	Resource *subsystems.ResourceConfig
}

func NewCgroupManager(path string) *CgroupManager {
	return &CgroupManager{
		Path: path,
	}
}

func (c *CgroupManager) Apply(pid int, res *subsystems.ResourceConfig) error {
	for _, subSysIns := range(subsystems.SubsystemsIns) {
		if err := subSysIns.Apply(c.Path, pid, res); err != nil {
			log.Errorf("Applying cgroup error: %v", err)
		}
	}
	return nil
}

func (c *CgroupManager) Set(res *subsystems.ResourceConfig) error {
	for _, subSysIns := range(subsystems.SubsystemsIns) {
		if err := subSysIns.Set(c.Path, res); err != nil {
			log.Errorf("Setting cgroup error: %v", err)
		}
	}
	return nil
}

func (c *CgroupManager) Destroy() error {
	for _, subSysIns := range(subsystems.SubsystemsIns) {
		if err := subSysIns.Remove(c.Path); err != nil {
			log.Warnf("remove cgroup fail %v", err)
		}
	}
	return nil
}
