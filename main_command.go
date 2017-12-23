package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/seagullbird/mydocker/container"
	"github.com/seagullbird/mydocker/cgroups/subsystems"
	"os"
	"strconv"
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
		cli.StringFlag{
			Name:	"v",
			Usage:	"volume",
		},
		cli.BoolFlag{
			Name:	"d",
			Usage:	"detach container",
		},
		cli.StringFlag{
			Name:	"name",
			Usage:	"container name",
		},
	},
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container command")
		}
		tty := context.Bool("it")
		detach := context.Bool("d")
		if tty && detach {
			return fmt.Errorf("-it and -d parameter can not both exist.")
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
		volume := context.String("v")
		containerName := context.String("name")
		Run(tty, cmdArray, resConf, volume, containerName)
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

var commitCommand = cli.Command{
	Name:  "commit",
	Usage: "commit a container into image",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container name")
		}
		imageName := context.Args().Get(0)
		//commitContainer(containerName)
		commitContainer(imageName)
		return nil
	},
}

var listCommand = cli.Command{
	Name: "ps",
	Usage: "list all the containers",
	Action: func(context *cli.Context) error {
		ListContainers()
		return nil
	},
}

var logCommand = cli.Command{
	Name: "logs",
	Usage: "print logs of a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Please input container name")
		}
		containerName := context.Args().Get(0)
		logContainer(containerName)
		return nil
	},
}

var execCommand = cli.Command{
	Name: "exec",
	Usage: "exec a command into a container",
	Action: func(context *cli.Context) error {
		// The second time it gots here, C code has already been executed, thus return
		if os.Getenv(ENV_EXEC_CMD) != "" {
			log.Infof("pid callback pid %s", strconv.Itoa(os.Getgid()))
			return nil
		}
		if len(context.Args()) < 2 {
			return fmt.Errorf("Missing container name or command")
		}
		containerName := context.Args().Get(0)
		var cmdArray []string
		for _, arg := range context.Args().Tail() {
			cmdArray = append(cmdArray, arg)
		}
		ExecContainer(containerName, cmdArray)
		return nil
	},
}

var stopCommand = cli.Command{
	Name: "stop",
	Usage: "stop a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container name")
		}
		containerName := context.Args().Get(0)
		stopContainer(containerName)
		return nil
	},
}

var removeCommand = cli.Command{
	Name: "rm",
	Usage: "remove a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container name")
		}
		containerName := context.Args().Get(0)
		removeContainer(containerName)
		return nil
	},
}