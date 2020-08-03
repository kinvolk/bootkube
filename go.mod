module github.com/kubernetes-sigs/bootkube

go 1.13

require (
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/groupcache v0.0.0-20191027212112-611e8accdfc9 // indirect
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.11.3 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/pborman/uuid v1.2.0
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/spf13/cobra v1.0.0
	go.etcd.io/bbolt v1.3.4 // indirect
	go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738
	go.uber.org/zap v1.14.1 // indirect
	golang.org/x/crypto v0.0.0-20200414173820-0848c9571904
	golang.org/x/net v0.0.0-20191109021931-daa7c04131f5
	golang.org/x/sys v0.0.0-20200202164722-d101bd2416d5
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	google.golang.org/genproto v0.0.0-20191108220845-16a3f7862a1a // indirect
	google.golang.org/grpc v1.27.0
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	helm.sh/helm/v3 v3.2.1
	k8s.io/api v0.18.2
	k8s.io/apiextensions-apiserver v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
	k8s.io/klog v1.0.0
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0
	// Pin to 1.26.0 due to https://github.com/etcd-io/etcd/issues/11563
	// `go mod tidy` without the below line update grpc to 1.27.0 which causes errors
	// make
	// mkdir -p _output/bin/linux/
	// GOOS=linux GOARCH=amd64   go build  -ldflags "-X github.com/kubernetes-sigs/bootkube/pkg/version.Version=319f07ab969aab0afbc466a8176681700b150bab-dirty" -o _output/bin/linux/bootkube github.com/kubernetes-sigs/bootkube/cmd/bootkube
	// # go.etcd.io/etcd/clientv3/balancer/picker
	// ../../../../pkg/mod/go.etcd.io/etcd@v0.0.0-20191023171146-3cf2f69b5738/clientv3/balancer/picker/err.go:37:44: undefined: balancer.PickOptions
	// ../../../../pkg/mod/go.etcd.io/etcd@v0.0.0-20191023171146-3cf2f69b5738/clientv3/balancer/picker/roundrobin_balanced.go:55:54: undefined: balancer.PickOptions
	// # go.etcd.io/etcd/clientv3/balancer/resolver/endpoint
	// ../../../../pkg/mod/go.etcd.io/etcd@v0.0.0-20191023171146-3cf2f69b5738/clientv3/balancer/resolver/endpoint/endpoint.go:114:78: undefined: resolver.BuildOption
	// ../../../../pkg/mod/go.etcd.io/etcd@v0.0.0-20191023171146-3cf2f69b5738/clientv3/balancer/resolver/endpoint/endpoint.go:182:31: undefined: resolver.ResolveNowOption
	// make: *** [Makefile:50: _output/bin/linux/bootkube] Error 2
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)
