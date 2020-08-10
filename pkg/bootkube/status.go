package bootkube

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/kubernetes-sigs/bootkube/pkg/util"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	doesNotExist = "DoesNotExist"
)

func WaitUntilPodsRunning(c clientcmd.ClientConfig, pods []string, timeout time.Duration) error {
	sc, err := NewStatusController(c, pods)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sc.Run(ctx)

	if err := wait.Poll(5*time.Second, timeout, sc.AllRunning); err != nil {
		return fmt.Errorf("error while checking pod status: %v", err)
	}

	util.UserOutput("All self-hosted control plane components successfully started\n")
	return nil
}

type statusController struct {
	client             kubernetes.Interface
	podStore           cache.Store
	nodeStore          cache.Store
	watchPods          []string
	lastPodPhases      map[string]corev1.PodPhase
	lastNodeConditions map[string]corev1.NodeCondition
}

func NewStatusController(c clientcmd.ClientConfig, pods []string) (*statusController, error) {
	config, err := c.ClientConfig()
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &statusController{client: client, watchPods: pods}, nil
}

func (s *statusController) Run(ctx context.Context) {
	// TODO: launch a separate watcher for each component?
	s.podWatcher(ctx)
	s.nodeWatcher(ctx)
}

func (s *statusController) podWatcher(ctx context.Context) {
	// TODO(yifan): Be more explicit about the labels so that we don't just
	// reply on the prefix of the pod name when looking for the pods we are interested.
	// E.g. For a scheduler pod, we will look for pods that has label `tier=control-plane`
	// and `component=kube-scheduler`.
	options := metav1.ListOptions{}
	podStore, podController := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(lo metav1.ListOptions) (runtime.Object, error) {
				return s.client.CoreV1().Pods("").List(ctx, options)
			},
			WatchFunc: func(lo metav1.ListOptions) (watch.Interface, error) {
				return s.client.CoreV1().Pods("").Watch(ctx, options)
			},
		},
		&corev1.Pod{},
		30*time.Minute,
		cache.ResourceEventHandlerFuncs{},
	)

	s.podStore = podStore
	go podController.Run(ctx.Done())
}

func (s *statusController) nodeWatcher(ctx context.Context) {
	options := metav1.ListOptions{}

	nodeStore, nodeController := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(lo metav1.ListOptions) (runtime.Object, error) {
				return s.client.CoreV1().Nodes().List(ctx, options)
			},
			WatchFunc: func(lo metav1.ListOptions) (watch.Interface, error) {
				return s.client.CoreV1().Nodes().Watch(ctx, options)
			},
		},
		&corev1.Node{},
		30*time.Minute,
		cache.ResourceEventHandlerFuncs{},
	)

	s.nodeStore = nodeStore

	go nodeController.Run(ctx.Done())
}
func (s *statusController) AllRunning() (bool, error) {
	var podsRunning, nodesReady, running bool

	podsRunning = s.allPodsRunning()

	if podsRunning {
		nodesReady = s.allNodesReady()
	}

	if podsRunning && nodesReady {
		running = true
	}

	return running, nil
}

func (s *statusController) allPodsRunning() bool {
	ps, err := s.PodStatus()
	if err != nil {
		glog.Infof("Error retriving pod statuses: %v", err)
		return false
	}

	if s.lastPodPhases == nil {
		s.lastPodPhases = ps
	}

	// use lastPodPhases to print only pods whose phase has changed
	changed := !reflect.DeepEqual(ps, s.lastPodPhases)
	s.lastPodPhases = ps

	running := true
	for p, s := range ps {
		if changed {
			util.UserOutput("\tPod Status:%24s\t%s\n", p, s)
		}
		if s != corev1.PodRunning {
			running = false
		}
	}

	return running
}

func (s *statusController) allNodesReady() bool {
	// Check node status to ensure all nodes are Ready
	ns, err := s.NodeStatus()
	if err != nil {
		glog.Infof("Error retrieving node conditions: %v", err)
		return false
	}

	if s.lastNodeConditions == nil {
		s.lastNodeConditions = ns
	}

	changed := !reflect.DeepEqual(ns, s.lastNodeConditions)
	s.lastNodeConditions = ns

	running := true
	for node, condition := range ns {
		if changed {
			util.UserOutput("\tNode Conditions:%24s\t%s\n", node, condition)
		}
		if condition.Status != corev1.ConditionTrue {
			running = false
		}
	}

	return running
}

func (s *statusController) PodStatus() (map[string]corev1.PodPhase, error) {
	status := make(map[string]corev1.PodPhase)

	podNames := s.podStore.ListKeys()
	for _, watchedPod := range s.watchPods {
		// Pod names are suffixed with random data. Match on prefix
		for _, pn := range podNames {
			if strings.HasPrefix(pn, watchedPod) {
				watchedPod = pn
				break
			}
		}
		p, exists, err := s.podStore.GetByKey(watchedPod)
		if err != nil {
			return nil, err
		}
		if !exists {
			status[watchedPod] = doesNotExist
			continue
		}
		if p, ok := p.(*corev1.Pod); ok {
			status[watchedPod] = p.Status.Phase
		}
	}
	return status, nil
}

func (s *statusController) NodeStatus() (map[string]corev1.NodeCondition, error) {
	status := make(map[string]corev1.NodeCondition)

	for _, node := range s.nodeStore.List() {
		for _, condition := range node.(*corev1.Node).Status.Conditions {
			if condition.Type == corev1.NodeReady {
				status[node.(*corev1.Node).Name] = condition
				break
			}
		}
	}

	return status, nil
}
