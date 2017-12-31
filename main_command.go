package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/seagullbird/mydocker/cgroups/subsystems"
	"github.com/seagullbird/mydocker/container"
	"github.com/seagullbird/mydocker/network"
	"github.com/urfave/cli"
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
			Name:  "m",
			Usage: "memory limit",
		},
		cli.StringFlag{
			Name:  "cpuset",
			Usage: "cpuset limit",
		},
		cli.StringFlag{
			Name:  "cpushare",
			Usage: "cpu share limit",
		},
		cli.BoolFlag{
			Name:  "d",
			Usage: "detach container",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "container name",
		},
		cli.StringFlag{
			Name:  "v",
			Usage: "volume",
		},
		cli.StringSliceFlag{
			Name:  "e",
			Usage: "set environment variables",
		},
		cli.StringFlag{
			Name:  "net",
			Usage: "container network",
		},
		cli.StringSliceFlag{
			Name:  "p",
			Usage: "port mapping",
		},
	},
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container command")
		}
		tty := context.Bool("it")
		memoryLimit := context.String("m")
		cpuset := context.String("cpuset")
		cpushare := context.String("cpushare")
		detach := context.Bool("d")
		containerName := context.String("name")
		volume := context.String("v")
		envSlice := context.StringSlice("e")
		network := context.String("net")
		portmapping := context.StringSlice("p")

		if tty && detach {
			return fmt.Errorf("-it and -d parameter can not both exist.")
		}
		resConf := &subsystems.ResourceConfig{
			MemoryLimit: memoryLimit,
			CpuSet:      cpuset,
			CpuShare:    cpushare,
		}
		var cmdArray []string
		for _, arg := range context.Args() {
			cmdArray = append(cmdArray, arg)
		}
		imageName := cmdArray[0]
		cmdArray = cmdArray[1:]
		Run(tty, cmdArray, resConf, volume, containerName, imageName, envSlice, network, portmapping)
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
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "package name",
		},
	},
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container name")
		}
		packageName := context.String("name")
		containerName := context.Args().Get(0)
		commitContainer(packageName, containerName)
		return nil
	},
}

var listCommand = cli.Command{
	Name:  "ps",
	Usage: "list all the containers",
	Action: func(context *cli.Context) error {
		ListContainers()
		return nil
	},
}

var logCommand = cli.Command{
	Name:  "logs",
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
	Name:  "exec",
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
	Name:  "stop",
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
	Name:  "rm",
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

var networkCommand = cli.Command{
	Name:  "network",
	Usage: "container network commmands",
	Subcommands: []cli.Command{
		{
			Name:  "create",
			Usage: "create a container network",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "driver",
					Usage: "Driver to manage the Network",
				},
				cli.StringFlag{
					Name:  "subnet",
					Usage: "Subnet in CIDR format that represents a network segment",
				},
			},
			Action: func(context *cli.Context) error {
				if len(context.Args()) < 1 {
					return fmt.Errorf("Missing network name")
				}
				driver := context.String("driver")
				subnet := context.String("subnet")
				networkName := context.Args().Get(0)
				network.Init()
				err := network.CreateNetwork(driver, subnet, networkName)
				if err != nil {
					return fmt.Errorf("Create network error: %v", err)
				}
				return nil
			},
		},
		{
			Name:  "list",
			Usage: "list container network",
			Action: func(context *cli.Context) error {
				network.Init()
				network.ListNetwork()
				return nil
			},
		},
		{
			Name:  "remove",
			Usage: "remove container network",
			Action: func(context *cli.Context) error {
				if len(context.Args()) < 1 {
					return fmt.Errorf("Missing network name")
				}
				network.Init()
				err := network.DeleteNetwork(context.Args().Get(0))
				if err != nil {
					return fmt.Errorf("Remove network error: %v", err)
				}
				return nil
			},
		},
	},
}
