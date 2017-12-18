package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/seagullbird/mydocker/container"
	"github.com/seagullbird/mydocker/cgroups/subsystems"
)

var runCommand = cli.Command{
	Name: "run",
	Usage: `Create a container with namespace and cgroup limit
			mydocker run -it [command]`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "it",
			Usage: "enable tty",
		},
		cli.StringFlag{
			Name:	"m",
			Usage:	"memory limit",
		},
		cli.StringFlag{
			Name:	"cpuset",
			Usage:	"cpuset limit",
		},
		cli.StringFlag{
			Name:	"cpushare",
			Usage:	"cpu share limit",
		},
	},
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container command")
		}
		var cmdArray []string
		for _, arg := range context.Args() {
			cmdArray = append(cmdArray, arg)
		}
		resConf := &subsystems.ResourceConfig{
			MemoryLimit:	context.String("m"),
			CpuSet:			context.String("cpuset"),
			CpuShare:		context.String("cpushare"),
		}
		tty := context.Bool("it")
		Run(tty, cmdArray, resConf)
		return nil
	},
}

var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside",
	Action: func(context *cli.Context) error {
		log.Infof("init come on")
		err := container.RunContainerInitProcess()
		return err
	},
}
