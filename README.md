# mydocker

*Learning Docker by making one.*

## Introduction

This is a simple prototype-level implementation of a container runtime engine.

It supports basic container managements including run, stop, remove, exec, commit, logs and so on.

Inter-container network, container-Internet network are also supported.

It does not and will not support image building.

Made for learning and fun.

## Contribute

You can clone this repo anytime and make pull requests.

After cloning, run:

```shell
$ go build . && go install
```

And the executive should be under ${GOPATH}/bin.

You can always run commands with `--help` when you don't know how to do stuff.

**!Important**

Before you run any containers, make sure the output of:

```shell
$ findmnt -o TARGET,PROPAGATION
```

is all **private**.

(Should look like this)

```shell
TARGET                                    PROPAGATION
/                                         private
├─/sys                                    private
│ ├─/sys/kernel/security                  private
│ ├─/sys/fs/cgroup                        private
│ │ ├─/sys/fs/cgroup/unified              private
│ │ ├─/sys/fs/cgroup/systemd              private
│ │ ├─/sys/fs/cgroup/net_cls,net_prio     private
│ │ ├─/sys/fs/cgroup/pids                 private
│ │ ├─/sys/fs/cgroup/cpuset               private
│ │ ├─/sys/fs/cgroup/perf_event           private
│ │ ├─/sys/fs/cgroup/cpu,cpuacct          private
│ │ ├─/sys/fs/cgroup/memory               private
│ │ ├─/sys/fs/cgroup/freezer              private
│ │ ├─/sys/fs/cgroup/blkio                private
│ │ ├─/sys/fs/cgroup/rdma                 private
│ │ ├─/sys/fs/cgroup/hugetlb              private
│ │ └─/sys/fs/cgroup/devices              private
│ ├─/sys/fs/pstore                        private
│ ├─/sys/kernel/debug                     private
│ ├─/sys/fs/fuse/connections              private
│ └─/sys/kernel/config                    private
├─/proc                                   private
│ └─/proc/sys/fs/binfmt_misc              private
├─/dev                                    private
│ ├─/dev/pts                              private
│ ├─/dev/shm                              private
│ ├─/dev/hugepages                        private
│ └─/dev/mqueue                           private
├─/run                                    private
│ ├─/run/lock                             private
│ ├─/run/vmblock-fuse                     private
│ ├─/run/user/0                           private
│ │ └─/run/user/0/gvfs                    private
│ ├─/run/netns                            private
│ └─/run/user/121                         private
├─/var/lib/docker/plugins                 private
├─/var/lib/docker/overlay2                private
└─/var/lib/mydocker/overlay2/redis/merged private
```

If not, run:

```shell
$ mount --make-rprivate /
```

to set all propagation type to *private*.

If you insist on running containers with a *shared* propagation type, make sure you made snapshots :)

## Images

*mydocker* does not support image management, but you can always use *Docker* to pull images.

To get a certain image, just run:

```shell
./pull_image.sh <image_name>[:<version>]
```

before running:

```shell
$ mydocker run -it <image_name> sh
```

Or whatever command you wish to start a container based on the image.

For more details checkout `pull_image.sh` :)

## Networking

Just remember to

1. set `net.ipv4.ip_forward=1`

2. *ACCEPT* packages in the *FORWARD* chain of the *filter* table of *iptables*.

```shell
$ iptables -P FORWARD ACCEPT
```
