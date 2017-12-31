package network

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/seagullbird/mydocker/container"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
)

type Network struct {
	// Name of the network
	Name string
	// the network's IP range (IP/mask)
	IpRange *net.IPNet
	// Name of the network driver
	Driver string
}

var (
	networkRootDir = filepath.Join(container.RootDir, "network")
	networkInfoDir = filepath.Join(networkRootDir, "networks")
	drivers        = map[string]NetworkDriver{}
	networks       = map[string]*Network{}
)

type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MACAddress  net.HardwareAddr `json:"mac"`
	PortMapping []string         `json:"portmapping"`
	Network     *Network
}

type NetworkDriver interface {
	// returns the name of the driver
	Name() string
	// create a network
	Create(subnet *net.IPNet, name string) (*Network, error)
	// delete a network
	Delete(network *Network) error
	// connect an endpoint to a network
	Connect(network *Network, endpoint *Endpoint) error
	// disconnect an endpoint from a network
	Disconnect(network *Network, endpoint *Endpoint) error
}

func CreateNetwork(driver, subnet, name string) error {
	// ipRange is a net.IPNet pointer
	_, ipRange, _ := net.ParseCIDR(subnet)

	// allocate an ip from the given network
	// use this ip as the gateway ip of the network
	gatewayIp, err := ipAllocator.Allocate(ipRange)
	if err != nil {
		return err
	}
	ipRange.IP = gatewayIp
	// now I have a gateway ip stored in ipRange.IP
	// as well as the network mask stored in ipRange.Mask
	// I can create a network using the given driver
	nw, err := drivers[driver].Create(ipRange, name)
	if err != nil {
		return err
	}
	// store the created network info
	return nw.dump(networkInfoDir)
}

func (nw *Network) dump(dumpPath string) error {
	if _, err := os.Stat(dumpPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(dumpPath, 0644)
		} else {
			return err
		}
	}
	nwPath := filepath.Join(dumpPath, nw.Name)
	nwFile, err := os.OpenFile(nwPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Errorf("Open nwfile %s error: %v", nwPath, err)
		return err
	}
	defer nwFile.Close()

	nwJson, err := json.Marshal(nw)
	if err != nil {
		log.Errorf("Marshal nw error: %v", err)
		return err
	}
	_, err = nwFile.Write(nwJson)
	if err != nil {
		log.Errorf("Write nw error: %v", err)
		return err
	}
	return nil
}

func (nw *Network) load(loadPath string) error {
	nwConfigFile, err := os.Open(loadPath)
	defer nwConfigFile.Close()
	if err != nil {
		return err
	}
	nwJson := make([]byte, 2000)
	n, err := nwConfigFile.Read(nwJson)
	if err != nil {
		return err
	}
	err = json.Unmarshal(nwJson[:n], nw)
	if err != nil {
		log.Errorf("Error load nw info from %s: %v", loadPath, err)
		return err
	}
	return nil
}

func (nw *Network) remove(dumpPath string) error {
	if _, err := os.Stat(filepath.Join(dumpPath, nw.Name)); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return os.Remove(filepath.Join(dumpPath, nw.Name))
}

func Connect(networkName string, cinfo *container.ContainerInfo) error {
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No such Network: %s", networkName)
	}
	// get ip address for the container
	ip, err := ipAllocator.Allocate(network.IpRange)
	if err != nil {
		return err
	}

	// create network endpoint
	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", cinfo.Id, networkName),
		IPAddress:   ip,
		Network:     network,
		PortMapping: cinfo.PortMapping,
	}
	if err = drivers[network.Driver].Connect(network, ep); err != nil {
		return err
	}
	if err = configEndpointIpAddressAndRoute(ep, cinfo); err != nil {
		return err
	}

	return configPortMapping(ep, cinfo)
}

func Init() error {
	// New a driver and save it in drivers
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver

	// Check networkInfoDir exists and create it
	if _, err := os.Stat(networkInfoDir); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(networkInfoDir, 0644)
		} else {
			return err
		}
	}

	// for each file (should be a network configuration file) under networkInfoDir,  load it into the memory
	// save each loaded network into networks
	filepath.Walk(networkInfoDir, func(nwPath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		_, nwname := path.Split(nwPath)
		nw := &Network{
			Name: nwname,
		}

		if err := nw.load(nwPath); err != nil {
			log.Errorf("error load network: %v", err)
		}

		networks[nwname] = nw
		return nil
	})
	return nil
}

func ListNetwork() {
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tIpRange\tDriver\n")
	for _, nw := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			nw.Name,
			nw.IpRange.String(),
			nw.Driver,
		)
	}

	if err := w.Flush(); err != nil {
		log.Errorf("Flush error: %v", err)
		return
	}
}

func DeleteNetwork(networkName string) error {
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No such network: %s", networkName)
	}
	_, cidr, _ := net.ParseCIDR(nw.IpRange.String())
	if err := ipAllocator.Delete(cidr); err != nil {
		return fmt.Errorf("Error Removing network ipam: %v", err)
	}

	if err := drivers[nw.Driver].Delete(nw); err != nil {
		return fmt.Errorf("Error removing network driver: %v", err)
	}

	return nw.remove(networkInfoDir)
}

func configEndpointIpAddressAndRoute(ep *Endpoint, cinfo *container.ContainerInfo) error {
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)
	}

	defer enterContainerNetns(&peerLink, cinfo)()

	interfaceIP := ep.Network.IpRange
	interfaceIP.IP = ep.IPAddress

	if err = setInterfaceIP(ep.Device.PeerName, interfaceIP); err != nil {
		return fmt.Errorf("%v,%s", ep.Network, err)
	}

	if err = setInterfaceUP(ep.Device.PeerName); err != nil {
		return err
	}

	if err = setInterfaceUP("lo"); err != nil {
		return err
	}

	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")

	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.IpRange.IP,
		Dst:       cidr,
	}

	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}

	return nil
}

func enterContainerNetns(enLink *netlink.Link, cinfo *container.ContainerInfo) func() {
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", cinfo.Pid), os.O_RDONLY, 0)
	if err != nil {
		log.Errorf("error get container net namespace, %v", err)
	}

	nsFD := f.Fd()
	runtime.LockOSThread()

	// 修改veth peer 另外一端移到容器的namespace中
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		log.Errorf("error set link netns , %v", err)
	}

	// 获取当前的网络namespace
	origns, err := netns.Get()
	if err != nil {
		log.Errorf("error get current netns, %v", err)
	}

	// 设置当前进程到新的网络namespace，并在函数执行完成之后再恢复到之前的namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		log.Errorf("error set netns, %v", err)
	}
	return func() {
		netns.Set(origns)
		origns.Close()
		runtime.UnlockOSThread()
		f.Close()
	}
}

func configPortMapping(ep *Endpoint, cinfo *container.ContainerInfo) error {
	for _, pm := range ep.PortMapping {
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			log.Errorf("port mapping format error, %v", pm)
			continue
		}
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			portMapping[0], ep.IPAddress.String(), portMapping[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		//err := cmd.Run()
		output, err := cmd.Output()
		if err != nil {
			log.Errorf("iptables Output, %v", output)
			continue
		}
	}
	return nil
}
