package network

import (
	"encoding/binary"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"net"
	"os"
	"path/filepath"
	"strings"
)

var ipamDefaultAllocatorPath = filepath.Join(networkRootDir, "ipam", "subnet.json")

type IPAM struct {
	subnetAllocatorPath string
	Subnets             *map[string]string
}

var ipAllocator = &IPAM{
	subnetAllocatorPath: ipamDefaultAllocatorPath,
}

func (ipam *IPAM) load() error {
	if _, err := os.Stat(ipam.subnetAllocatorPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	subnetConfigFile, err := os.Open(ipam.subnetAllocatorPath)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}
	subnetJson := make([]byte, 2000)
	n, err := subnetConfigFile.Read(subnetJson)
	if err != nil {
		return err
	}

	err = json.Unmarshal(subnetJson[:n], ipam.Subnets)
	if err != nil {
		log.Errorf("Error loading allocation info: %v", err)
		return err
	}
	return nil
}

func (ipam *IPAM) dump() error {
	ipamConfigFileDir, _ := filepath.Split(ipam.subnetAllocatorPath)
	if _, err := os.Stat(ipamConfigFileDir); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(ipamConfigFileDir, 0644)
		} else {
			return err
		}
	}
	subnetConfigFile, err := os.OpenFile(ipam.subnetAllocatorPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}
	ipamConfigJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return err
	}
	_, err = subnetConfigFile.Write(ipamConfigJson)
	if err != nil {
		log.Errorf("Error dumping allocation info: %v", err)
		return err
	}
	return nil
}

// Google "golang int to ip"
func ip2int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func int2ip(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}

func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	_, cidr, _ := net.ParseCIDR(subnet.String())
	ipam.Subnets = &map[string]string{}

	err = ipam.load()
	if err != nil {
		log.Errorf("Error in loading allocation info: %v", err)
	}
	ones, bits := subnet.Mask.Size()

	if _, exists := (*ipam.Subnets)[cidr.String()]; !exists {
		(*ipam.Subnets)[cidr.String()] = strings.Repeat("0", 1<<uint8(bits-ones))
	}
	for c := range (*ipam.Subnets)[cidr.String()] {
		if (*ipam.Subnets)[cidr.String()][c] == '0' {
			ipalloc := []byte((*ipam.Subnets)[cidr.String()])
			ipalloc[c] = '1'
			(*ipam.Subnets)[cidr.String()] = string(ipalloc)
			ip = cidr.IP

			ipint := ip2int(ip)
			ip = int2ip(ipint + uint32(c) + 1)
			break
		}
	}

	ipam.dump()
	return
}

func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr net.IP) (err error) {
	_, cidr, _ := net.ParseCIDR(subnet.String())
	ipam.Subnets = &map[string]string{}
	err = ipam.load()
	if err != nil {
		log.Errorf("Error in loading allocation info: %v", err)
	}
	c := ip2int(ipaddr) - ip2int(cidr.IP) - 1
	ipalloc := []byte((*ipam.Subnets)[cidr.String()])
	ipalloc[c] = '0'
	(*ipam.Subnets)[cidr.String()] = string(ipalloc)
	ipam.dump()
	return
}

func (ipam *IPAM) Delete(subnet *net.IPNet) error {
	ipam.Subnets = &map[string]string{}
	err := ipam.load()
	if err != nil {
		log.Errorf("Error in loading allocation info: %v", err)
	}
	delete(*ipam.Subnets, subnet.String())
	return ipam.dump()
}
