package network

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/seagullbird/mydocker/container"
	"github.com/vishvananda/netlink"
	"net"
	"os"
	"path"
	"path/filepath"
	"text/tabwriter"
)

type Network struct {
	Name    string
	IpRange *net.IPNet
	Driver  string
}

var (
	networkInfoDir = "/var/lib/mydocker/network"
	drivers        = map[string]NetworkDriver{}
	networks       = map[string]*Network{}
)

type Endpoint struct {
	ID          string           `json:id`
	Device      netlink.Veth     `json:dev`
	IPAddress   net.IP           `json:ip`
	MACAddress  net.HardwareAddr `json:mac`
	PortMapping []string         `json:portmapping`
	Network     *Network
}

type NetworkDriver interface {
	Name() string
	Create(subnet, name string) (*Network, error)
	Delete(network *Network) error
	Connect(network *Network, endpoint *Endpoint) error
	Disconnect(network *Network, endpoint *Endpoint) error
}

func CreateNetwork(driver, subnet, name string) error {
	_, ipnet, _ := net.ParseCIDR(subnet)
	gatewayIp, err := ipAllocator.Allocate(ipnet)
	if err != nil {
		return err
	}
	ipnet.IP = gatewayIp
	nw, err := drivers[driver].Create(ipnet.String(), name)
	if err != nil {
		return err
	}
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
	nwPath := path.Join(dumpPath, nw.Name)
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
	if _, err := os.Stat(path.Join(dumpPath, nw.Name)); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return os.Remove(path.Join(dumpPath, nw.Name))
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
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver

	if _, err := os.Stat(networkInfoDir); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(networkInfoDir, 0644)
		} else {
			return err
		}
	}

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
	if err := ipAllocator.Release(nw.IpRange, &nw.IpRange.IP); err != nil {
		return fmt.Errorf("Error Removing network Driver: %v", err)
	}

	return nw.remove(networkInfoDir)
}
