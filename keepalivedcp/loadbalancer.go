package keepalivedcp

import (
	"fmt"
	"net"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/cloudprovider"
)

const configMapAnnotationKey = "k8s.co/cloud-provider-config"

type KeepalivedLoadBalancer struct {
	kubeClient      *kubernetes.Clientset
	namespace, name string
	serviceCidr     string
}

var _ cloudprovider.LoadBalancer = &KeepalivedLoadBalancer{}

func NewKeepalivedLoadBalancer(kubeClient *kubernetes.Clientset, ns, name, serviceCidr string) cloudprovider.LoadBalancer {
	return &KeepalivedLoadBalancer{kubeClient, ns, name, serviceCidr}
}

func (k *KeepalivedLoadBalancer) GetLoadBalancer(clusterName string, service *v1.Service) (status *v1.LoadBalancerStatus, exists bool, err error) {
	cm, err := k.getConfigMap()

	if err != nil {
		return nil, false, err
	}

	cfg, err := configFrom(cm)

	if err != nil {
		return nil, false, err
	}

	for _, svc := range cfg.services {
		if svc.uid == string(service.UID) {
			return &v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{{IP: svc.ip}},
			}, true, nil
		}
	}

	return nil, false, nil
}

func (k *KeepalivedLoadBalancer) EnsureLoadBalancer(clusterName string, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
	return k.syncLoadBalancer(service)
}

func (k *KeepalivedLoadBalancer) UpdateLoadBalancer(clusterName string, service *v1.Service, nodes []*v1.Node) error {
	_, err := k.syncLoadBalancer(service)
	return err
}

func (k *KeepalivedLoadBalancer) EnsureLoadBalancerDeleted(clusterName string, service *v1.Service) error {
	return k.deleteLoadBalancer(service)
}

func (k *KeepalivedLoadBalancer) deleteLoadBalancer(service *v1.Service) error {
	cm, err := k.getConfigMap()

	if err != nil {
		return err
	}

	cfg, err := configFrom(cm)

	if err != nil {
		return err
	}

	for _, svc := range cfg.services {
		// service already exists in the config so just return the status
		if svc.uid == string(service.UID) {
			cfg.deleteService(svc)
			delete(cm.Data, svc.ip)

			if _, err = k.kubeClient.ConfigMaps(k.namespace).Update(cm); err != nil {
				return fmt.Errorf("error updating keepalived config: %s", err.Error())
			}

			return nil
		}
	}

	return nil
}

func (k *KeepalivedLoadBalancer) syncLoadBalancer(service *v1.Service) (*v1.LoadBalancerStatus, error) {
	cm, err := k.getConfigMap()

	if err != nil {
		return nil, err
	}

	cfg, err := configFrom(cm)

	if err != nil {
		return nil, err
	}

	for _, svc := range cfg.services {
		// service already exists in the config so just return the status
		if svc.uid == string(service.UID) {
			// if there's a mismatch between desired loadBalancerIP and actual,
			// break out of this loop and continue to update
			if service.Spec.LoadBalancerIP != svc.ip {
				break
			}

			return &v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{{IP: svc.ip}},
			}, nil
		}
	}

	var ip string
	if lbip := service.Spec.LoadBalancerIP; lbip != "" {
		if i := net.ParseIP(lbip); i == nil {
			return nil, fmt.Errorf("invalid loadBalancerIP specified '%s': %s", lbip, err.Error())
		}
		ip = lbip
	} else {
		ip, err = cfg.allocateIP(k.serviceCidr)
		if err != nil {
			return nil, err
		}
	}

	cfg.ensureService(serviceConfig{uid: string(service.UID), ip: ip})
	cfgBytes, err := cfg.encode()

	if err != nil {
		return nil, fmt.Errorf("error encoding updated config: %s", err.Error())
	}

	cm.Data[ip] = service.Namespace + "/" + service.Name
	cm.Annotations[configMapAnnotationKey] = string(cfgBytes)

	if _, err = k.kubeClient.ConfigMaps(k.namespace).Update(cm); err != nil {
		return nil, fmt.Errorf("error updating keepalived config: %s", err.Error())
	}

	return &v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{{IP: ip}},
	}, nil
}

func (k *KeepalivedLoadBalancer) getConfigMap() (*apiv1.ConfigMap, error) {
	cm, err := k.kubeClient.ConfigMaps(k.namespace).Get(k.name, metav1.GetOptions{})

	if err != nil {
		return nil, fmt.Errorf("error getting keepalived configmap: %s", err.Error())
	}

	return cm, err
}
