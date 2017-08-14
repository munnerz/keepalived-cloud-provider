package keepalivedcp

import (
	"fmt"
	"io"

	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/cloudprovider"
)

const (
	ProviderName = "keepalived"
)

func init() {
	cloudprovider.RegisterCloudProvider(ProviderName, newKeepalivedCloudProvider)
}

type KeepalivedCloudProvider struct {
	lb cloudprovider.LoadBalancer
}

var _ cloudprovider.Interface = &KeepalivedCloudProvider{}

func newKeepalivedCloudProvider(io.Reader) (cloudprovider.Interface, error) {
	ns := os.Getenv("KEEPALIVED_NAMESPACE")
	cm := os.Getenv("KEEPALIVED_CONFIG_MAP")
	cidr := os.Getenv("KEEPALIVED_SERVICE_CIDR")

	cfg, err := rest.InClusterConfig()

	if err != nil {
		return nil, fmt.Errorf("error creating kubernetes client config: %s", err.Error())
	}

	cl, err := kubernetes.NewForConfig(cfg)

	if err != nil {
		return nil, fmt.Errorf("error creating kubernetes client: %s", err.Error())
	}

	return &KeepalivedCloudProvider{NewKeepalivedLoadBalancer(cl, ns, cm, cidr)}, nil
}

// LoadBalancer returns a loadbalancer interface. Also returns true if the interface is supported, false otherwise.
func (k *KeepalivedCloudProvider) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return k.lb, true
}

// Instances returns an instances interface. Also returns true if the interface is supported, false otherwise.
func (k *KeepalivedCloudProvider) Instances() (cloudprovider.Instances, bool) {
	return nil, false
}

// Zones returns a zones interface. Also returns true if the interface is supported, false otherwise.
func (k *KeepalivedCloudProvider) Zones() (cloudprovider.Zones, bool) {
	return zones{}, true
}

// Clusters returns a clusters interface.  Also returns true if the interface is supported, false otherwise.
func (k *KeepalivedCloudProvider) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

// Routes returns a routes interface along with whether the interface is supported.
func (k *KeepalivedCloudProvider) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}

// ProviderName returns the cloud provider ID.
func (k *KeepalivedCloudProvider) ProviderName() string {
	return "keepalived"
}

// ScrubDNS provides an opportunity for cloud-provider-specific code to process DNS settings for pods.
func (k *KeepalivedCloudProvider) ScrubDNS(nameservers, searches []string) (nsOut, srchOut []string) {
	return nil, nil
}

type zones struct{}

func (z zones) GetZone() (cloudprovider.Zone, error) {
	return cloudprovider.Zone{FailureDomain: "FailureDomain1", Region: "Region1"}, nil
}
