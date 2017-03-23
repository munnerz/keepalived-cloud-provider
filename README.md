# keepalived-cloud-provider

keepalived-cloud-provider is a Kubernetes cloud provider implementation (more info: https://github.com/wlan0/kubernetes.github.io/blob/c0f3aa4abe99ad0528c6ec168e8fbf14fdaf49ac/docs/getting-started-guides/running-cloud-controller.md).

It allows users in bare-metal environments to use services with `type: LoadBalancer` set, automatically provisioning an IP address within the provided `KEEPALIVED_SERVICE_CIDR` and writing a [kube-keepalived-vip](https://github.com/kubernetes/contrib/tree/master/keepalived-vip) ConfigMap. Thus, this controller is not responsible for the load balancing itself.

This is perfect if you want to run Kubernetes in network in which you have local, routable access to a given CIDR.

An image of this project exists (although is not currently versioned) at: `eu.gcr.io/marley-xyz/keepalived-cloud-provider`
