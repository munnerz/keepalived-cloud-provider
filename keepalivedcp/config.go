package keepalivedcp

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/golang/glog"

	"k8s.io/client-go/pkg/api/v1"
)

type config struct {
	services []serviceConfig
}

func (c *config) allocateIP(cidr string) (string, error) {
	possible, err := Hosts(cidr)
	if err != nil {
		return "", err
	}

	for _, ip := range possible {
		for _, svc := range c.services {
			// if this 'ip' candidate is already in use,
			// break the inner loop to move onto the next IP address
			if svc.ip == ip {
				break
			}
		}

		// if we get to this point, then 'ip' hasn't been allocated already
		return ip, nil
	}

	return "", fmt.Errorf("ip cidr pool exhausted. increase size of cidr or remove some loadbalancers")
}

func (c *config) encode() ([]byte, error) {
	return json.Marshal(c)
}

func (c *config) ensureService(cfg serviceConfig) {
	for i, s := range c.services {
		if s.uid == cfg.uid {
			glog.Infof("updating service with uid '%s' in config: %s->%s", cfg.uid, s.ip, cfg.ip)
			c.services[i] = cfg
			return
		}
	}
	glog.Infof("adding new service '%s': %s", cfg.uid, cfg.ip)
	c.services = append(c.services, cfg)
	glog.Infof("there are now %d services in config", len(c.services))
}

func (c *config) deleteService(cfg serviceConfig) {
	for i, s := range c.services {
		if s.uid == cfg.uid {
			glog.Infof("deleted service with uid %s, ip: %s from config", s.uid, s.ip)
			c.services = append(c.services[:i], c.services[i+1:]...)
			return
		}
	}
}

type serviceConfig struct {
	uid string
	ip  string
}

func configFrom(cm *v1.ConfigMap) (*config, error) {
	cfg := config{}
	if c, ok := cm.Annotations[configMapAnnotationKey]; ok {
		err := json.Unmarshal([]byte(c), &cfg)

		if err != nil {
			return nil, fmt.Errorf("error getting cloud provider config from annotation: %s", err.Error())
		}
	}
	return &cfg, nil
}

// from: https://gist.github.com/kotakanbe/d3059af990252ba89a82
func Hosts(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}
	// remove network address and broadcast address
	return ips[1 : len(ips)-1], nil
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
