package helm

import (
	"fmt"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"log"

	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	//	"os"
	"time"
)

func debug(format string, v ...interface{}) {
	format = fmt.Sprintf("[debug] %s\n", format)
	log.Output(2, fmt.Sprintf(format, v...))
}

func apiTest(c clientcmd.ClientConfig) error {
	config, err := c.ClientConfig()
	if err != nil {
		return err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// API Server is responding
	healthStatus := 0
	client.Discovery().RESTClient().Get().AbsPath("/healthz").Do().StatusCode(&healthStatus)
	if healthStatus != http.StatusOK {
		return fmt.Errorf("API Server http status: %d", healthStatus)
	}

	// System namespace has been created
	_, err = client.CoreV1().Namespaces().Get("kube-system", metav1.GetOptions{})
	return err
}

func InstallHelmChart(config clientcmd.ClientConfig, manifestDir string, timeout time.Duration, strict bool) error {
	// if _, err := os.Stat(manifestDir); os.IsNotExist(err) {
	// 	UserOutput(fmt.Sprintf("WARNING: %v does not exist, not creating any self-hosted assets.\n", manifestDir))
	// 	return nil
	// }

	upFn := func() (bool, error) {
		if err := apiTest(config); err != nil {
			glog.Warningf("Unable to determine api-server readiness: %v", err)
			return false, nil
		}
		return true, nil
	}

	UserOutput("Waiting for api-server...\n")
	if err := wait.Poll(5*time.Second, timeout, upFn); err != nil {
		err = fmt.Errorf("API Server is not ready: %v", err)
		glog.Error(err)
		return err
	}

	client := NewClientGetter(config)

	actionConfig := &action.Configuration{}

	actionConfig.RESTClientGetter = client
	if err := actionConfig.Init(client, "kube-system", "secrets", debug); err != nil {
		return err
	}

	// LoadChart from the manifest directory
	ch, err := loader.Load(manifestDir)
	if err != nil {
		return err
	}

	install := action.NewInstall(actionConfig)
	install.ReleaseName = "kubernetes-components"
	install.Namespace = "kube-system"

	release, err := install.Run(ch, ch.Values)
	if err != nil {
		return err
	}
	// Load values.yaml file as a string
	// Later we can template it out as it is lokoctl components
	//

	UserOutput(fmt.Sprintf("Release-created :: %s", release.Name))

	return nil
}

// All bootkube printing to stdout should go through this fmt.Printf wrapper.
// The stdout of bootkube should convey information useful to a human sitting
// at a terminal watching their cluster bootstrap itself. Otherwise the message
// should go to stderr.
func UserOutput(format string, a ...interface{}) {
	fmt.Printf(format, a...)
}

func LoadChart(name string) (*chart.Chart, error) {

	ch, err := loader.Load(name)
	if err != nil {
		return nil, err
	}

	return ch, err
	// Load Chart
}
