// Package internal holds asset templates used by bootkube.
package internal

var AdminKubeConfigTemplate = []byte(`apiVersion: v1
kind: Config
clusters:
- name: local
  cluster:
    server: {{ .Server }}
    certificate-authority-data: {{ .CACert }}
users:
- name: admin
  user:
    client-certificate-data: {{ .AdminCert }}
    client-key-data: {{ .AdminKey }}
contexts:
- context:
    cluster: local
    user: admin
`)

var KubeletKubeConfigTemplate = []byte(`apiVersion: v1
kind: Config
clusters:
- name: local
  cluster:
    server: {{ .Server }}
    certificate-authority-data: {{ .CACert }}
users:
- name: kubelet
  user:
    token: {{ .BootstrapTokenID }}.{{ .BootstrapTokenSecret }}
contexts:
- context:
    cluster: local
    user: kubelet
`)

var KubeletBootstrappingToken = []byte(`apiVersion: v1
kind: Secret
metadata:
  name: bootstrap-token-{{ .BootstrapTokenID }}
  namespace: kube-system
type: bootstrap.kubernetes.io/token
stringData:
  token-id: "{{ .BootstrapTokenID }}"
  token-secret: "{{ .BootstrapTokenSecret }}"
  usage-bootstrap-authentication: "true"
`)

// CSRNodeBootstrapTemplate lets bootstrapping tokens and nodes request CSRs.
var CSRNodeBootstrapTemplate = []byte(`kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system-bootstrap-node-bootstrapper
subjects:
- kind: Group
  name: system:bootstrappers
  apiGroup: rbac.authorization.k8s.io
- kind: Group
  name: system:nodes
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: system:node-bootstrapper
  apiGroup: rbac.authorization.k8s.io
`)

// CSRApproverRoleBindingTemplate instructs the csrapprover controller to
// automatically approve CSRs made by bootstrapping tokens for client
// credentials.
//
// This binding should be removed to disable CSR auto-approval.
var CSRApproverRoleBindingTemplate = []byte(`kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system-bootstrap-approve-node-client-csr
subjects:
- kind: Group
  name: system:bootstrappers
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: system:certificates.k8s.io:certificatesigningrequests:nodeclient
  apiGroup: rbac.authorization.k8s.io
`)

// CSRRenewalRoleBindingTemplate instructs the csrapprover controller to
// automatically approve all CSRs made by nodes to renew their client
// certificates.
//
// This binding should be altered in the future to hold a list of node
// names instead of targeting `system:nodes` so we can revoke invidivual
// node's ability to renew its certs.
var CSRRenewalRoleBindingTemplate = []byte(`kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system-bootstrap-node-renewal
subjects:
- kind: Group
  name: system:nodes
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: system:certificates.k8s.io:certificatesigningrequests:selfnodeclient
  apiGroup: rbac.authorization.k8s.io
`)

var KubeSystemSARoleBindingTemplate = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:default-sa
subjects:
  - kind: ServiceAccount
    name: default
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
`)

var APIServerTemplate = []byte(`apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: kube-apiserver
  namespace: kube-system
  labels:
    tier: control-plane
    k8s-app: kube-apiserver
spec:
  selector:
    matchLabels:
      tier: control-plane
      k8s-app: kube-apiserver
  template:
    metadata:
      labels:
        tier: control-plane
        k8s-app: kube-apiserver
      annotations:
        checkpointer.alpha.coreos.com/checkpoint: "true"
    spec:
      containers:
      - name: kube-apiserver
        image: {{ .Images.Hyperkube }}
        command:
        - /hyperkube
        - kube-apiserver
        - --enable-admission-plugins=NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeClaimResize,DefaultStorageClass,DefaultTolerationSeconds,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota,Priority,NodeRestriction
        - --advertise-address=$(POD_IP)
        - --allow-privileged=true
        - --anonymous-auth=false
        - --authorization-mode=Node,RBAC
        - --bind-address=0.0.0.0
        - --client-ca-file=/etc/kubernetes/secrets/ca.crt
        - --requestheader-client-ca-file=/etc/kubernetes/secrets/front-proxy-ca.crt
        - --requestheader-allowed-names=front-proxy-client
        - --requestheader-extra-headers-prefix=X-Remote-Extra-
        - --requestheader-group-headers=X-Remote-Group
        - --requestheader-username-headers=X-Remote-User
        - --proxy-client-cert-file=/etc/kubernetes/secrets/front-proxy-client.crt
        - --proxy-client-key-file=/etc/kubernetes/secrets/front-proxy-client.key
        - --cloud-provider={{ .CloudProvider }}
        - --enable-bootstrap-token-auth=true
{{- if .EtcdUseTLS }}
        - --etcd-cafile=/etc/kubernetes/secrets/etcd-client-ca.crt
        - --etcd-certfile=/etc/kubernetes/secrets/etcd-client.crt
        - --etcd-keyfile=/etc/kubernetes/secrets/etcd-client.key
{{- end }}
        - --etcd-servers={{ range $i, $e := .EtcdServers }}{{ if $i }},{{end}}{{ $e }}{{end}}
        - --insecure-port=0
        - --kubelet-client-certificate=/etc/kubernetes/secrets/apiserver-kubelet-client.crt
        - --kubelet-client-key=/etc/kubernetes/secrets/apiserver-kubelet-client.key
        - --secure-port={{ (index .APIServers 0).Port }}
        - --service-account-key-file=/etc/kubernetes/secrets/service-account.pub
        - --service-cluster-ip-range={{ .ServiceCIDR }}
        - --tls-cert-file=/etc/kubernetes/secrets/apiserver.crt
        - --tls-private-key-file=/etc/kubernetes/secrets/apiserver.key
        env:
        - name: POD_IP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        volumeMounts:
        - mountPath: /etc/ssl/certs
          name: ssl-certs-host
          readOnly: true
        - mountPath: /etc/kubernetes/secrets
          name: secrets
          readOnly: true
      hostNetwork: true
      nodeSelector:
        node-role.kubernetes.io/master: ""
      tolerations:
      - key: node-role.kubernetes.io/master
        operator: Exists
        effect: NoSchedule
      volumes:
      - name: ssl-certs-host
        hostPath:
          path: /usr/share/ca-certificates
      - name: secrets
        secret:
          secretName: kube-apiserver
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
  updateStrategy:
    rollingUpdate:
      maxUnavailable: 1
    type: RollingUpdate
`)

var BootstrapAPIServerTemplate = []byte(`apiVersion: v1
kind: Pod
metadata:
  name: bootstrap-kube-apiserver
  namespace: kube-system
spec:
  containers:
  - name: kube-apiserver
    image: {{ .Images.Hyperkube }}
    command:
    - /hyperkube
    - kube-apiserver
    - --advertise-address=$(POD_IP)
    - --allow-privileged=true
    - --authorization-mode=Node,RBAC
    - --bind-address=0.0.0.0
    - --client-ca-file=/etc/kubernetes/secrets/ca.crt
    - --requestheader-client-ca-file=/etc/kubernetes/secrets/front-proxy-ca.crt
    - --requestheader-allowed-names=front-proxy-client
    - --requestheader-extra-headers-prefix=X-Remote-Extra-
    - --requestheader-group-headers=X-Remote-Group
    - --requestheader-username-headers=X-Remote-User
    - --proxy-client-cert-file=/etc/kubernetes/secrets/front-proxy-client.crt
    - --proxy-client-key-file=/etc/kubernetes/secrets/front-proxy-client.key
    - --enable-admission-plugins=NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeClaimResize,DefaultStorageClass,DefaultTolerationSeconds,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota,Priority,NodeRestriction
    - --enable-bootstrap-token-auth=true
{{- if .EtcdUseTLS }}
    - --etcd-cafile=/etc/kubernetes/secrets/etcd-client-ca.crt
    - --etcd-certfile=/etc/kubernetes/secrets/etcd-client.crt
    - --etcd-keyfile=/etc/kubernetes/secrets/etcd-client.key
{{- end }}
    - --etcd-servers={{ range $i, $e := .EtcdServers }}{{ if $i }},{{end}}{{ $e }}{{end}}
    - --kubelet-client-certificate=/etc/kubernetes/secrets/apiserver-kubelet-client.crt
    - --kubelet-client-key=/etc/kubernetes/secrets/apiserver-kubelet-client.key
    - --secure-port={{ (index .APIServers 0).Port }}
    - --service-account-key-file=/etc/kubernetes/secrets/service-account.pub
    - --service-cluster-ip-range={{ .ServiceCIDR }}
    - --cloud-provider={{ .CloudProvider }}
    - --tls-cert-file=/etc/kubernetes/secrets/apiserver.crt
    - --tls-private-key-file=/etc/kubernetes/secrets/apiserver.key
    env:
    - name: POD_IP
      valueFrom:
        fieldRef:
          fieldPath: status.podIP
    volumeMounts:
    - mountPath: /etc/ssl/certs
      name: ssl-certs-host
      readOnly: true
    - mountPath: /etc/kubernetes/secrets
      name: secrets
      readOnly: true
  hostNetwork: true
  volumes:
  - name: secrets
    hostPath:
      path: /etc/kubernetes/{{ .BootstrapSecretsSubdir }}
  - name: ssl-certs-host
    hostPath:
      path: /usr/share/ca-certificates
`)

var CheckpointerTemplate = []byte(`apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: pod-checkpointer
  namespace: kube-system
  labels:
    tier: control-plane
    k8s-app: pod-checkpointer
spec:
  selector:
    matchLabels:
      tier: control-plane
      k8s-app: pod-checkpointer
  template:
    metadata:
      labels:
        tier: control-plane
        k8s-app: pod-checkpointer
      annotations:
        checkpointer.alpha.coreos.com/checkpoint: "true"
    spec:
      containers:
      - name: pod-checkpointer
        image: {{ .Images.PodCheckpointer }}
        command:
        - /checkpoint
        - --lock-file=/var/run/lock/pod-checkpointer.lock
        - --kubeconfig=/etc/checkpointer/kubeconfig
        - --checkpoint-grace-period=5m
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        imagePullPolicy: Always
        volumeMounts:
        - mountPath: /etc/checkpointer
          name: kubeconfig
        - mountPath: /etc/kubernetes
          name: etc-kubernetes
        - mountPath: /var/run
          name: var-run
      serviceAccountName: pod-checkpointer
      hostNetwork: true
      nodeSelector:
        node-role.kubernetes.io/master: ""
      restartPolicy: Always
      tolerations:
      - key: node-role.kubernetes.io/master
        operator: Exists
        effect: NoSchedule
      volumes:
      - name: kubeconfig
        configMap:
          name: kubeconfig-in-cluster
      - name: etc-kubernetes
        hostPath:
          path: /etc/kubernetes
      - name: var-run
        hostPath:
          path: /var/run
  updateStrategy:
    rollingUpdate:
      maxUnavailable: 1
    type: RollingUpdate
`)

var CheckpointerServiceAccount = []byte(`apiVersion: v1
kind: ServiceAccount
metadata:
  namespace: kube-system
  name: pod-checkpointer
`)

var CheckpointerRole = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: pod-checkpointer
  namespace: kube-system
rules:
- apiGroups: [""] # "" indicates the core API group
  resources: ["pods"]
  verbs: ["get", "watch", "list"]
- apiGroups: [""] # "" indicates the core API group
  resources: ["secrets", "configmaps"]
  verbs: ["get"]
`)

var CheckpointerRoleBinding = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: pod-checkpointer
  namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: pod-checkpointer
subjects:
- kind: ServiceAccount
  name: pod-checkpointer
  namespace: kube-system
`)

var CheckpointerClusterRole = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: pod-checkpointer
rules:
  - apiGroups: [""]
    resources: ["nodes", "nodes/proxy"]
    verbs: ["get"]
`)

var CheckpointerClusterRoleBinding = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: pod-checkpointer
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: pod-checkpointer
subjects:
- kind: ServiceAccount
  name: pod-checkpointer
  namespace: kube-system
`)

var ControllerManagerTemplate = []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-controller-manager
  namespace: kube-system
  labels:
    tier: control-plane
    k8s-app: kube-controller-manager
spec:
  replicas: 2
  selector:
    matchLabels:
      tier: control-plane
      k8s-app: kube-controller-manager
  template:
    metadata:
      labels:
        tier: control-plane
        k8s-app: kube-controller-manager
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: tier
                  operator: In
                  values:
                  - control-plane
                - key: k8s-app
                  operator: In
                  values:
                  - kube-controller-manager
              topologyKey: kubernetes.io/hostname
      containers:
      - name: kube-controller-manager
        image: {{ .Images.Hyperkube }}
        command:
        - ./hyperkube
        - kube-controller-manager
        - --use-service-account-credentials
        - --allocate-node-cidrs=true
        - --cloud-provider={{ .CloudProvider }}
        - --cluster-cidr={{ .PodCIDR }}
        - --service-cluster-ip-range={{ .ServiceCIDR }}
        - --cluster-signing-cert-file=/etc/kubernetes/secrets/ca.crt
        - --cluster-signing-key-file=/etc/kubernetes/secrets/ca.key
        - --configure-cloud-routes=false
        - --leader-elect=true
        - --root-ca-file=/etc/kubernetes/secrets/ca.crt
        - --service-account-private-key-file=/etc/kubernetes/secrets/service-account.key
        livenessProbe:
          httpGet:
            path: /healthz
            port: 10252  # Note: Using default port. Update if --port option is set differently.
          initialDelaySeconds: 15
          timeoutSeconds: 15
        volumeMounts:
        - name: var-run-kubernetes
          mountPath: /var/run/kubernetes
        - name: secrets
          mountPath: /etc/kubernetes/secrets
          readOnly: true
        - name: ssl-host
          mountPath: /etc/ssl/certs
          readOnly: true
      nodeSelector:
        node-role.kubernetes.io/master: ""
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
      serviceAccountName: kube-controller-manager
      tolerations:
      - key: node-role.kubernetes.io/master
        operator: Exists
        effect: NoSchedule
      volumes:
      - name: var-run-kubernetes
        emptyDir: {}
      - name: secrets
        secret:
          secretName: kube-controller-manager
      - name: ssl-host
        hostPath:
          path: /usr/share/ca-certificates
      dnsPolicy: ClusterFirstWithHostNet
`)

var ControllerManagerServiceAccount = []byte(`apiVersion: v1
kind: ServiceAccount
metadata:
  namespace: kube-system
  name: kube-controller-manager
`)

var ControllerManagerClusterRoleBinding = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: controller-manager
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:kube-controller-manager
subjects:
- kind: ServiceAccount
  name: kube-controller-manager
  namespace: kube-system
`)

var BootstrapControllerManagerTemplate = []byte(`apiVersion: v1
kind: Pod
metadata:
  name: bootstrap-kube-controller-manager
  namespace: kube-system
spec:
  containers:
  - name: kube-controller-manager
    image: {{ .Images.Hyperkube }}
    command:
    - ./hyperkube
    - kube-controller-manager
    - --allocate-node-cidrs=true
    - --cluster-cidr={{ .PodCIDR }}
    - --service-cluster-ip-range={{ .ServiceCIDR }}
    - --cloud-provider={{ .CloudProvider }}
    - --cluster-signing-cert-file=/etc/kubernetes/secrets/ca.crt
    - --cluster-signing-key-file=/etc/kubernetes/secrets/ca.key
    - --configure-cloud-routes=false
    - --kubeconfig=/etc/kubernetes/secrets/kubeconfig
    - --leader-elect=true
    - --root-ca-file=/etc/kubernetes/secrets/ca.crt
    - --service-account-private-key-file=/etc/kubernetes/secrets/service-account.key
    volumeMounts:
    - name: secrets
      mountPath: /etc/kubernetes/secrets
      readOnly: true
    - name: ssl-host
      mountPath: /etc/ssl/certs
      readOnly: true
  hostNetwork: true
  volumes:
  - name: secrets
    hostPath:
      path: /etc/kubernetes/{{ .BootstrapSecretsSubdir }}
  - name: ssl-host
    hostPath:
      path: /usr/share/ca-certificates
`)

var ControllerManagerDisruptionTemplate = []byte(`apiVersion: policy/v1beta1
kind: PodDisruptionBudget
metadata:
  name: kube-controller-manager
  namespace: kube-system
spec:
  minAvailable: 1
  selector:
    matchLabels:
      tier: control-plane
      k8s-app: kube-controller-manager
`)

var SchedulerTemplate = []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-scheduler
  namespace: kube-system
  labels:
    tier: control-plane
    k8s-app: kube-scheduler
spec:
  replicas: 2
  selector:
    matchLabels:
      tier: control-plane
      k8s-app: kube-scheduler
  template:
    metadata:
      labels:
        tier: control-plane
        k8s-app: kube-scheduler
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: tier
                  operator: In
                  values:
                  - control-plane
                - key: k8s-app
                  operator: In
                  values:
                  - kube-scheduler
              topologyKey: kubernetes.io/hostname
      containers:
      - name: kube-scheduler
        image: {{ .Images.Hyperkube }}
        command:
        - ./hyperkube
        - kube-scheduler
        - --leader-elect=true
        livenessProbe:
          httpGet:
            path: /healthz
            port: 10251  # Note: Using default port. Update if --port option is set differently.
          initialDelaySeconds: 15
          timeoutSeconds: 15
      nodeSelector:
        node-role.kubernetes.io/master: ""
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
      tolerations:
      - key: node-role.kubernetes.io/master
        operator: Exists
        effect: NoSchedule
`)

var BootstrapSchedulerTemplate = []byte(`apiVersion: v1
kind: Pod
metadata:
  name: bootstrap-kube-scheduler
  namespace: kube-system
spec:
  containers:
  - name: kube-scheduler
    image: {{ .Images.Hyperkube }}
    command:
    - ./hyperkube
    - kube-scheduler
    - --kubeconfig=/etc/kubernetes/secrets/kubeconfig
    - --leader-elect=true
    volumeMounts:
    - name: secrets
      mountPath: /etc/kubernetes/secrets
      readOnly: true
  hostNetwork: true
  volumes:
  - name: secrets
    hostPath:
      path: /etc/kubernetes/{{ .BootstrapSecretsSubdir }}
`)

var SchedulerDisruptionTemplate = []byte(`apiVersion: policy/v1beta1
kind: PodDisruptionBudget
metadata:
  name: kube-scheduler
  namespace: kube-system
spec:
  minAvailable: 1
  selector:
    matchLabels:
      tier: control-plane
      k8s-app: kube-scheduler
`)

var ProxyTemplate = []byte(`apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: kube-proxy
  namespace: kube-system
  labels:
    tier: node
    k8s-app: kube-proxy
spec:
  selector:
    matchLabels:
      tier: node
      k8s-app: kube-proxy
  template:
    metadata:
      labels:
        tier: node
        k8s-app: kube-proxy
    spec:
      containers:
      - name: kube-proxy
        image: {{ .Images.Hyperkube }}
        command:
        - ./hyperkube
        - kube-proxy
        - --cluster-cidr={{ .PodCIDR }}
        - --hostname-override=$(NODE_NAME)
        - --kubeconfig=/etc/kubernetes/kubeconfig
        - --proxy-mode=iptables
        env:
          - name: NODE_NAME
            valueFrom:
              fieldRef:
                fieldPath: spec.nodeName
        securityContext:
          privileged: true
        volumeMounts:
        - mountPath: /lib/modules
          name: lib-modules
          readOnly: true
        - mountPath: /etc/ssl/certs
          name: ssl-certs-host
          readOnly: true
        - name: kubeconfig
          mountPath: /etc/kubernetes
          readOnly: true
      hostNetwork: true
      serviceAccountName: kube-proxy
      tolerations:
      - effect: NoSchedule
        operator: Exists
      - effect: NoExecute
        operator: Exists
      volumes:
      - name: lib-modules
        hostPath:
          path: /lib/modules
      - name: ssl-certs-host
        hostPath:
          path: /usr/share/ca-certificates
      - name: kubeconfig
        configMap:
          name: kubeconfig-in-cluster
  updateStrategy:
    rollingUpdate:
      maxUnavailable: 1
    type: RollingUpdate
`)

var ProxyServiceAccount = []byte(`apiVersion: v1
kind: ServiceAccount
metadata:
  namespace: kube-system
  name: kube-proxy
`)

var ProxyClusterRoleBinding = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kube-proxy
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:node-proxier # Automatically created system role.
subjects:
- kind: ServiceAccount
  name: kube-proxy
  namespace: kube-system
`)

// KubeConfigInCluster instructs clients to use their service account token,
// but unlike an in-cluster client doesn't rely on the `KUBERNETES_SERVICE_PORT`
// and `KUBERNETES_PORT` to determine the API servers address.
//
// This kubeconfig is used by bootstrapping pods that might not have access to
// these env vars, such as kube-proxy, which sets up the API server endpoint
// (chicken and egg), and the checkpointer, which needs to run as a static pod
// even if the API server isn't available.
var KubeConfigInClusterTemplate = []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: kubeconfig-in-cluster
  namespace: kube-system
data:
  kubeconfig: |
    apiVersion: v1
    clusters:
    - name: local
      cluster:
        server: {{ .Server }}
        certificate-authority: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    users:
    - name: service-account
      user:
        # Use service account token
        tokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
    contexts:
    - context:
        cluster: local
        user: service-account
`)

var CoreDNSClusterRoleBindingTemplate = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:coredns
  labels:
    kubernetes.io/bootstrapping: rbac-defaults
  annotations:
    rbac.authorization.kubernetes.io/autoupdate: "true"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:coredns
subjects:
  - kind: ServiceAccount
    name: coredns
    namespace: kube-system
`)

var CoreDNSClusterRoleTemplate = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: system:coredns
  labels:
    kubernetes.io/bootstrapping: rbac-defaults
rules:
  - apiGroups: [""]
    resources:
      - endpoints
      - services
      - pods
      - namespaces
    verbs:
      - list
      - watch
  - apiGroups: [""]
    resources:
      - nodes
    verbs:
      - get
`)

var CoreDNSConfigTemplate = []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |
    .:53 {
        errors
        health {
          lameduck 5s
        }
        ready
        log . {
            class error
        }
        kubernetes cluster.local in-addr.arpa ip6.arpa {
            pods insecure
            fallthrough in-addr.arpa ip6.arpa
        }
        prometheus :9153
        forward . /etc/resolv.conf
        cache 30
        loop
        reload
        loadbalance
    }
`)

var CoreDNSDeploymentTemplate = []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: coredns
  namespace: kube-system
  labels:
    k8s-app: kube-dns
    kubernetes.io/name: "CoreDNS"
    kubernetes.io/cluster-service: "true"
spec:
  replicas : 2
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  selector:
    matchLabels:
      k8s-app: coredns
  template:
    metadata:
      labels:
        k8s-app: coredns
      annotations:
        seccomp.security.alpha.kubernetes.io/pod: 'docker/default'
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: k8s-app
                  operator: In
                  values:
                  - coredns
              topologyKey: kubernetes.io/hostname
      serviceAccountName: coredns
      tolerations:
        - key: node-role.kubernetes.io/master
          effect: NoSchedule
      containers:
        - name: coredns
          image: {{ .Images.CoreDNS }}
          imagePullPolicy: IfNotPresent
          resources:
            limits:
              memory: 170Mi
            requests:
              cpu: 100m
              memory: 70Mi
          args: [ "-conf", "/etc/coredns/Corefile" ]
          volumeMounts:
            - name: config-volume
              mountPath: /etc/coredns
              readOnly: true
          ports:
            - name: dns
              protocol: UDP
              containerPort: 53
            - name: dns-tcp
              protocol: TCP
              containerPort: 53
            - name: metrics
              protocol: TCP
              containerPort: 9153
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
              scheme: HTTP
            initialDelaySeconds: 60
            timeoutSeconds: 5
            successThreshold: 1
            failureThreshold: 5
          readinessProbe:
            httpGet:
              path: /ready
              port: 8181
              scheme: HTTP
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              add:
              - NET_BIND_SERVICE
              drop:
              - all
            readOnlyRootFilesystem: true
      dnsPolicy: Default
      volumes:
        - name: config-volume
          configMap:
            name: coredns
            items:
            - key: Corefile
              path: Corefile
`)

var CoreDNSServiceAccountTemplate = []byte(`apiVersion: v1
kind: ServiceAccount
metadata:
  name: coredns
  namespace: kube-system
  labels:
    kubernetes.io/cluster-service: "true"
`)

var CoreDNSSvcTemplate = []byte(`apiVersion: v1
kind: Service
metadata:
  name: kube-dns
  namespace: kube-system
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "9153"
  labels:
    k8s-app: kube-dns
    kubernetes.io/cluster-service: "true"
    kubernetes.io/name: "CoreDNS"
spec:
  selector:
    k8s-app: kube-dns
  clusterIP: {{ .DNSServiceIP }}
  ports:
    - name: dns
      port: 53
      protocol: UDP
    - name: dns-tcp
      port: 53
      protocol: TCP
`)

var FlannelClusterRole = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: flannel
rules:
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
  - apiGroups:
      - ""
    resources:
      - nodes
    verbs:
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - nodes/status
    verbs:
      - patch
`)

var FlannelClusterRoleBinding = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: flannel
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: flannel
subjects:
- kind: ServiceAccount
  name: flannel
  namespace: kube-system
`)

var FlannelServiceAccount = []byte(`apiVersion: v1
kind: ServiceAccount
metadata:
  name: flannel
  namespace: kube-system
`)

var FlannelCfgTemplate = []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: kube-flannel-cfg
  namespace: kube-system
  labels:
    tier: node
    k8s-app: flannel
data:
  cni-conf.json: |
    {
      "name": "cbr0",
      "cniVersion": "0.3.1",
      "plugins": [
        {
          "type": "flannel",
          "delegate": {
            "hairpinMode": true,
            "isDefaultGateway": true
          }
        },
        {
          "type": "portmap",
          "capabilities": {
            "portMappings": true
          }
        }
      ]
    }
  net-conf.json: |
    {
      "Network": "{{ .PodCIDR }}",
      "Backend": {
        "Type": "vxlan",
        "Port": 4789
      }
    }
`)

var FlannelTemplate = []byte(`apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: kube-flannel
  namespace: kube-system
  labels:
    tier: node
    k8s-app: flannel
spec:
  selector:
    matchLabels:
      tier: node
      k8s-app: flannel
  template:
    metadata:
      labels:
        tier: node
        k8s-app: flannel
    spec:
      serviceAccountName: flannel
      containers:
      - name: kube-flannel
        image: {{ .Images.Flannel }}
        command: [ "/opt/bin/flanneld", "--ip-masq", "--kube-subnet-mgr", "--iface=$(POD_IP)"]
        securityContext:
          privileged: true
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: POD_IP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        volumeMounts:
        - name: run
          mountPath: /run
        - name: cni
          mountPath: /etc/cni/net.d
        - name: flannel-cfg
          mountPath: /etc/kube-flannel/
      - name: install-cni
        image: {{ .Images.FlannelCNI }}
        command: ["/install-cni.sh"]
        env:
        - name: CNI_NETWORK_CONFIG
          valueFrom:
            configMapKeyRef:
              name: kube-flannel-cfg
              key: cni-conf.json
        volumeMounts:
        - name: cni
          mountPath: /host/etc/cni/net.d
        - name: host-cni-bin
          mountPath: /host/opt/cni/bin/
      hostNetwork: true
      tolerations:
      - effect: NoSchedule
        operator: Exists
      - effect: NoExecute
        operator: Exists
      volumes:
        - name: run
          hostPath:
            path: /run
        - name: cni
          hostPath:
            path: /etc/kubernetes/cni/net.d
        - name: flannel-cfg
          configMap:
            name: kube-flannel-cfg
        - name: host-cni-bin
          hostPath:
            path: /opt/cni/bin
  updateStrategy:
    rollingUpdate:
      maxUnavailable: 1
    type: RollingUpdate
`)

var CalicoCfgTemplate = []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: calico-config
  namespace: kube-system
data:
  # Disable Typha for now.
  typha_service_name: "none"
  # Calico backend to use
  calico_backend: "bird"
  # Calico MTU
  veth_mtu: "1500"
  # The CNI network configuration to install on each node.
  cni_network_config: |-
    {
      "name": "k8s-pod-network",
      "cniVersion": "0.3.1",
      "plugins": [
        {
          "type": "calico",
          "log_level": "info",
          "datastore_type": "kubernetes",
          "nodename": "__KUBERNETES_NODE_NAME__",
          "mtu": __CNI_MTU__,
          "ipam": {
            "type": "calico-ipam"
          },
          "policy": {
            "type": "k8s"
          },
          "kubernetes": {
            "kubeconfig": "__KUBECONFIG_FILEPATH__"
          }
        },
        {
          "type": "portmap",
          "snat": true,
          "capabilities": {"portMappings": true}
        },
        {
          "type": "bandwidth",
          "capabilities": {"bandwidth": true}
        }

      ]
    }
`)

// calico template
var CalicoNodeTemplate = []byte(`apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: calico-node
  namespace: kube-system
  labels:
    k8s-app: calico-node
spec:
  selector:
    matchLabels:
      k8s-app: calico-node
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  template:
    metadata:
      labels:
        k8s-app: calico-node
      annotations:
        seccomp.security.alpha.kubernetes.io/pod: 'docker/default'
    spec:
      hostNetwork: true
      priorityClassName: system-node-critical
      serviceAccountName: calico-node
      tolerations:
        - effect: NoSchedule
          operator: Exists
        - effect: NoExecute
          operator: Exists
      initContainers:
        # Install Calico CNI binaries and CNI network config file on nodes
        - name: install-cni
          image: {{ .Images.Calico }}
          command: ["/install-cni.sh"]
          env:
            # Name of the CNI config file to create on each node.
            - name: CNI_CONF_NAME
              value: "10-calico.conflist"
            # Set node name based on k8s nodeName
            - name: KUBERNETES_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            # Contents of the CNI config to create on each node.
            - name: CNI_NETWORK_CONFIG
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: cni_network_config
            - name: CNI_NET_DIR
              value: "/etc/kubernetes/cni/net.d"
            - name: CNI_MTU
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: veth_mtu
            - name: SLEEP
              value: "false"
          volumeMounts:
            - name: cni-bin-dir
              mountPath: /host/opt/cni/bin
            - name: cni-conf-dir
              mountPath: /host/etc/cni/net.d
      containers:
        - name: calico-node
          image: {{ .Images.Calico }}
          env:
            # Use Kubernetes API as the backing datastore.
            - name: DATASTORE_TYPE
              value: "kubernetes"
            # Wait for datastore
            - name: WAIT_FOR_DATASTORE
              value: "true"
            # Typha support: controlled by the ConfigMap.
            - name: FELIX_TYPHAK8SSERVICENAME
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: typha_service_name
            - name: FELIX_USAGEREPORTINGENABLED
              value: "false"
            # Set node name based on k8s nodeName.
            - name: NODENAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            # Calico network backend
            - name: CALICO_NETWORKING_BACKEND
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: calico_backend
            # Cluster type to identify the deployment type
            - name: CLUSTER_TYPE
              value: "k8s,bgp"
            # Auto-detect the BGP IP address.
            - name: IP
              value: "autodetect"
            - name: IP_AUTODETECTION_METHOD
              value: "first-found"
            # Enable IP-in-IP within Felix.
            - name: FELIX_IPINIPENABLED
              value: "true"
            # Set MTU for tunnel device used if ipip is enabled
            - name: FELIX_IPINIPMTU
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: veth_mtu
            - name: NO_DEFAULT_POOLS
              value: "true"
            # Disable file logging so "kubectl logs" works.
            - name: CALICO_DISABLE_FILE_LOGGING
              value: "true"
            # Set Felix endpoint to host default action to ACCEPT.
            - name: FELIX_DEFAULTENDPOINTTOHOSTACTION
              value: "ACCEPT"
            # Disable IPV6 on Kubernetes.
            - name: FELIX_IPV6SUPPORT
              value: "false"
            # Enable felix info logging.
            - name: FELIX_LOGSEVERITYSCREEN
              value: "info"
            - name: FELIX_HEALTHENABLED
              value: "true"
          securityContext:
            privileged: true
          resources:
            requests:
              cpu: 150m
          livenessProbe:
            exec:
              command:
                - /bin/calico-node
                - -felix-ready
            periodSeconds: 10
            initialDelaySeconds: 10
            failureThreshold: 6
          readinessProbe:
            exec:
              command:
                - /bin/calico-node
                - -felix-ready
                - -bird-ready
            periodSeconds: 10
          volumeMounts:
            - name: lib-modules
              mountPath: /lib/modules
              readOnly: true
            - name: var-lib-calico
              mountPath: /var/lib/calico
              readOnly: false
            - name: var-run-calico
              mountPath: /var/run/calico
              readOnly: false
            - name: xtables-lock
              mountPath: /run/xtables.lock
              readOnly: false
      terminationGracePeriodSeconds: 0
      volumes:
        # Used by calico/node
        - name: lib-modules
          hostPath:
            path: /lib/modules
        - name: var-lib-calico
          hostPath:
            path: /var/lib/calico
        - name: var-run-calico
          hostPath:
            path: /var/run/calico
        - name: xtables-lock
          hostPath:
            type: FileOrCreate
            path: /run/xtables.lock
        # Used by install-cni
        - name: cni-bin-dir
          hostPath:
            path: /opt/cni/bin
        - name: cni-conf-dir
          hostPath:
            path: /etc/kubernetes/cni/net.d
`)

// Canal template
var CalicoPolicyOnlyTemplate = []byte(`apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: calico-node
  namespace: kube-system
  labels:
    k8s-app: calico-node
spec:
  selector:
    matchLabels:
      k8s-app: calico-node
  template:
    metadata:
      labels:
        k8s-app: calico-node
    spec:
      hostNetwork: true
      serviceAccountName: calico-node
      tolerations:
        - effect: NoSchedule
          operator: Exists
        - effect: NoExecute
          operator: Exists
      containers:
        - name: calico-node
          image: {{ .Images.Calico }}
          env:
            - name: DATASTORE_TYPE
              value: "kubernetes"
            - name: FELIX_LOGSEVERITYSCREEN
              value: "info"
            - name: CALICO_NETWORKING_BACKEND
              value: "none"
            - name: CLUSTER_TYPE
              value: "bootkube,canal"
            - name: CALICO_DISABLE_FILE_LOGGING
              value: "true"
            - name: FELIX_DEFAULTENDPOINTTOHOSTACTION
              value: "ACCEPT"
            - name: FELIX_IPV6SUPPORT
              value: "false"
            - name: WAIT_FOR_DATASTORE
              value: "true"
            - name: CALICO_IPV4POOL_CIDR
              value: "{{ .PodCIDR }}"
            - name: CALICO_IPV4POOL_IPIP
              value: "Always"
            - name: NODENAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: IP
              value: ""
            - name: FELIX_HEALTHENABLED
              value: "true"
          securityContext:
            privileged: true
          resources:
            requests:
              cpu: 250m
          livenessProbe:
            httpGet:
              path: /liveness
              port: 9099
            periodSeconds: 10
            initialDelaySeconds: 10
            failureThreshold: 6
          readinessProbe:
            httpGet:
              path: /readiness
              port: 9099
            periodSeconds: 10
          volumeMounts:
            - mountPath: /lib/modules
              name: lib-modules
              readOnly: true
            - mountPath: /var/run/calico
              name: var-run-calico
              readOnly: false
        - name: install-cni
          image: {{ .Images.CalicoCNI }}
          command: ["/install-cni.sh"]
          env:
            - name: CNI_CONF_NAME
              value: "10-calico.conflist"
            - name: CNI_NETWORK_CONFIG
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: cni_network_config
            - name: CNI_NET_DIR
              value: "/etc/kubernetes/cni/net.d"
            - name: KUBERNETES_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: SKIP_CNI_BINARIES
              value: bridge,cnitool,dhcp,flannel,host-local,ipvlan,loopback,macvlan,noop,portmap,ptp,tuning
          volumeMounts:
            - mountPath: /host/opt/cni/bin
              name: cni-bin-dir
            - mountPath: /host/etc/cni/net.d
              name: cni-net-dir
      terminationGracePeriodSeconds: 0
      volumes:
        - name: lib-modules
          hostPath:
            path: /lib/modules
        - name: var-run-calico
          hostPath:
            path: /var/run/calico
        - name: cni-bin-dir
          hostPath:
            path: /opt/cni/bin
        - name: cni-net-dir
          hostPath:
            path: /etc/kubernetes/cni/net.d
  updateStrategy:
    rollingUpdate:
      maxUnavailable: 1
    type: RollingUpdate
`)

var CalicoGlobalNetworkPoliciesCRD = []byte(`apiVersion: apiextensions.k8s.io/v1beta1
description: Calico Global Network Policies
kind: CustomResourceDefinition
metadata:
  name: globalnetworkpolicies.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: GlobalNetworkPolicy
    plural: globalnetworkpolicies
    singular: globalnetworkpolicy
`)

var CalicoIPPoolsCRD = []byte(`apiVersion: apiextensions.k8s.io/v1beta1
description: Calico IP Pools
kind: CustomResourceDefinition
metadata:
  name: ippools.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: IPPool
    plural: ippools
    singular: ippool
`)

var CalicoOtherCRD = []byte(`apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: blockaffinities.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: BlockAffinity
    plural: blockaffinities
    singular: blockaffinity
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: hostendpoints.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: HostEndpoint
    plural: hostendpoints
    singular: hostendpoint
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: ipamblocks.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: IPAMBlock
    plural: ipamblocks
    singular: ipamblock
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: ipamconfigs.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: IPAMConfig
    plural: ipamconfigs
    singular: ipamconfig
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: ipamhandles.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: IPAMHandle
    plural: ipamhandles
    singular: ipamhandle
---
`)

var CalicoBGPConfigurationsCRD = []byte(`apiVersion: apiextensions.k8s.io/v1beta1
description: Calico BGP Configuration
kind: CustomResourceDefinition
metadata:
  name: bgpconfigurations.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: BGPConfiguration
    plural: bgpconfigurations
    singular: bgpconfiguration
`)

var CalicoBGPPeersCRD = []byte(`apiVersion: apiextensions.k8s.io/v1beta1
description: Calico BGP Peers
kind: CustomResourceDefinition
metadata:
  name: bgppeers.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: BGPPeer
    plural: bgppeers
    singular: bgppeer
`)

var CalicoFelixConfigurationsCRD = []byte(`apiVersion: apiextensions.k8s.io/v1beta1
description: Calico Felix Configuration
kind: CustomResourceDefinition
metadata:
  name: felixconfigurations.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: FelixConfiguration
    plural: felixconfigurations
    singular: felixconfiguration
`)

var CalicoGlobalNetworkSetsCRD = []byte(`apiVersion: apiextensions.k8s.io/v1beta1
description: Calico Global Network Sets
kind: CustomResourceDefinition
metadata:
  name: globalnetworksets.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: GlobalNetworkSet
    plural: globalnetworksets
    singular: globalnetworkset
`)

var CalicoNetworkPoliciesCRD = []byte(`apiVersion: apiextensions.k8s.io/v1beta1
description: Calico Network Policies
kind: CustomResourceDefinition
metadata:
  name: networkpolicies.crd.projectcalico.org
spec:
  scope: Namespaced
  group: crd.projectcalico.org
  version: v1
  names:
    kind: NetworkPolicy
    plural: networkpolicies
    singular: networkpolicy
`)

var CalicoClusterInformationsCRD = []byte(`apiVersion: apiextensions.k8s.io/v1beta1
description: Calico Cluster Information
kind: CustomResourceDefinition
metadata:
  name: clusterinformations.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: ClusterInformation
    plural: clusterinformations
    singular: clusterinformation
`)

var CalicoServiceAccountTemplate = []byte(`apiVersion: v1
kind: ServiceAccount
metadata:
  name: calico-node
  namespace: kube-system
`)

var CalicoRoleTemplate = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: calico-node
rules:
  - apiGroups: [""]
    resources:
      - pods
      - nodes
      - namespaces
    verbs:
      - get
  - apiGroups: [""]
    resources:
      - endpoints
      - services
    verbs:
      - watch
      - list
  # Used by Calico for policy information
  - apiGroups: [""]
    resources:
      - pods
      - namespaces
      - serviceaccounts
    verbs:
      - list
      - watch
  - apiGroups: [""]
    resources:
      - nodes/status
    verbs:
      # Calico patches the node NetworkUnavilable status
      - patch
      # Calico updates some info in node annotations
      - update
  # CNI plugin patches pods/status
  - apiGroups: [""]
    resources:
      - pods/status
    verbs:
      - patch
  # Calico reads some info on nodes
  - apiGroups: [""]
    resources:
      - nodes
    verbs:
      - get
      - list
      - watch
  # Calico monitors Kubernetes NetworkPolicies
  - apiGroups: ["networking.k8s.io"]
    resources:
      - networkpolicies
    verbs:
      - watch
      - list
  # Calico monitors its CRDs
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - globalfelixconfigs
      - felixconfigurations
      - bgppeers
      - globalbgpconfigs
      - bgpconfigurations
      - ippools
      - ipamblocks
      - globalnetworkpolicies
      - globalnetworksets
      - networkpolicies
      - networksets
      - clusterinformations
      - hostendpoints
      - blockaffinities
    verbs:
      - get
      - list
      - watch
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - felixconfigurations
      - ippools
      - clusterinformations
    verbs:
      - create
      - update
  # Calico may perform IPAM allocations (not yet used)
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - blockaffinities
      - ipamblocks
      - ipamhandles
    verbs:
      - get
      - list
      - create
      - update
      - delete
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - ipamconfigs
    verbs:
      - get
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - blockaffinities
    verbs:
      - watch
`)

var CalicoRoleBindingTemplate = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: calico-node
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: calico-node
subjects:
- kind: ServiceAccount
  name: calico-node
  namespace: kube-system
`)

// vim: set expandtab:tabstop=2
