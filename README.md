# keepalived-cloud-provider [![Build Status](https://travis-ci.org/munnerz/keepalived-cloud-provider.svg?branch=master)](https://travis-ci.org/munnerz/keepalived-cloud-provider)

> This project is in alpha state, and should be used with caution. Whilst it is quite simple, there
> are currently minimal unit tests and no integration tests. Contributions are very welcome.

keepalived-cloud-provider is an out-of-tree Kubernetes cloud provider implementation ([more info](https://github.com/wlan0/kubernetes.github.io/blob/c0f3aa4abe99ad0528c6ec168e8fbf14fdaf49ac/docs/getting-started-guides/running-cloud-controller.md)).
It will manage and automatically update a ConfigMap for [kube-keepalived-vip](https://github.com/kubernetes/contrib/tree/master/keepalived-vip), which will then
automatically create load balanced IP addresses in the specified CIDR.
This allows users in bare-metal environments to use services with `type: LoadBalancer` set.

This is perfect if you want to run Kubernetes in network in which you have a routable CIDR
that you want to expose your services in.

## Getting started

To use the cloud provider, we'll need to do a few things:

- Install `kube-keepalived-vip`
- Set `--cloud-provider=external` on our kube-controller-manager master component
- Deploy `keepalived-cloud-provider`
- Create a service with `type: LoadBalancer`!

### Install kube-keepalived-vip

Full instructions are available in the `kube-keepalived-vip` [repository](https://github.com/kubernetes/contrib/tree/master/keepalived-vip).

Briefly, we simply need to create a DaemonSet:

```bash
$ kubectl create -f vip-daemonset.yaml
```

```yaml
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: kube-keepalived-vip
  namespace: kube-system
spec:
  template:
    metadata:
      labels:
        name: kube-keepalived-vip
    spec:
      hostNetwork: true
      containers:
        - image: gcr.io/google_containers/kube-keepalived-vip:0.9
          name: kube-keepalived-vip
          imagePullPolicy: Always
          securityContext:
            privileged: true
          volumeMounts:
            - mountPath: /lib/modules
              name: modules
              readOnly: true
            - mountPath: /dev
              name: dev
          # use downward API
          env:
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
          # to use unicast
          args:
          - --services-configmap=kube-system/vip-configmap
          # unicast uses the ip of the nodes instead of multicast
          # this is useful if running in cloud providers (like AWS)
          #- --use-unicast=true
      volumes:
        - name: modules
          hostPath:
            path: /lib/modules
        - name: dev
          hostPath:
            path: /dev
      nodeSelector:
        # type: worker # adjust this to match your worker nodes
---
## We also create an empty ConfigMap to hold our config
apiVersion: v1
kind: ConfigMap
metadata:
  name: vip-configmap
  namespace: kube-system
data:
```

### Configure kube-controller-manager

In order to use the currently alpha external cloud provider functionality, we need to set a
flag on the `kube-controller-manager` component. How to do this depends on how
you deployed your cluster, but if deployed with `kubeadm` you should edit
`/etc/kubernetes/manifests/kube-controller-manager.yaml`. and add
`--cloud-provider external` to the command section.

If you are using the `kubeadm` config file, then the following fragment will
enable the external cloud provider.

```yaml
controllerManagerExtraArgs:
  cloud-provider: external
```

### Deploy keepalived-cloud-provider

`keepalived-cloud-provider` can be deployed with a simple Kubernetes Deployment, and performs
leader election like other kubernetes master components. It is therefore safe to run multiple
replicas of the `keepalived-cloud-provider` pod.

```yaml
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  labels:
    app: keepalived-cloud-provider
  name: keepalived-cloud-provider
  namespace: kube-system
spec:
  replicas: 1
  revisionHistoryLimit: 2
  selector:
    matchLabels:
      app: keepalived-cloud-provider
  strategy:
    type: RollingUpdate
  template:
    metadata:
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ""
        scheduler.alpha.kubernetes.io/tolerations: '[{"key":"CriticalAddonsOnly", "operator":"Exists"}]'
      labels:
        app: keepalived-cloud-provider
    spec:
      containers:
      - name: keepalived-cloud-provider
        image: quay.io/munnerz/keepalived-cloud-provider:0.0.1
        imagePullPolicy: IfNotPresent
        env:
        - name: KEEPALIVED_NAMESPACE
          value: kube-system
        - name: KEEPALIVED_CONFIG_MAP
          value: vip-configmap
        - name: KEEPALIVED_SERVICE_CIDR
          value: 10.210.38.100/26 # pick a CIDR that is explicitly reserved for keepalived
        volumeMounts:
        - name: certs
          mountPath: /etc/ssl/certs
        resources:
          requests:
            cpu: 200m
        livenessProbe:
          httpGet:
            path: /healthz
            port: 10252
            host: 127.0.0.1
          initialDelaySeconds: 15
          timeoutSeconds: 15
          failureThreshold: 8
      volumes:
      - name: certs
        hostPath:
          path: /etc/ssl/certs
```

### Create a service

Once `keepalived-cloud-provider` is up and running, you should be able to create service with `type: LoadBalancer`:

```bash
$ kubectl expose deployment example-com --name=example-com --type=LoadBalancer
```

`keepalived-cloud-provider` will also honour the `loadBalancerIp` field in a `service.spec`, and will configure
a load balancer with the provided IP regardless whether it is within the `KEEPALIVED_SERVICE_CIDR`

```bash
$ kubectl get services
NAME              CLUSTER-IP       EXTERNAL-IP    PORT(S)        AGE
test              10.98.31.230     10.210.38.66   80:31877/TCP   3s
test2             10.107.177.153   10.210.38.65   80:31261/TCP   12m
```
