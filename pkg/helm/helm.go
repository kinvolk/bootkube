package helm

import (
	"fmt"
	"os"
	"path/filepath"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubernetes-sigs/bootkube/pkg/util"
)

const (
	defaultHelmStorageDriver = "secrets"
)

// InstallCharts installs all the helm charts in the given charts directory.
func InstallCharts(kubeconfigPath string, config clientcmd.ClientConfig, chartsDir string) error {
	// check if charts directory exists
	present, err := isExists(chartsDir)
	if err != nil {
		// error checking for the existence charts directory
		return err
	}
	// charts directory not found, nothing to do.
	if !present {
		return nil
	}
	// get all the charts
	charts, err := getCharts(chartsDir)
	if err != nil {
		return fmt.Errorf("getting charts from charts directory %q: %v", chartsDir, err)
	}
	// iterate over all the namespaces found in the charts directory
	for _, chart := range charts {
		chartPath := filepath.Join(chartsDir, chart.namespace, chart.name)
		// install charts found in each namespace directory
		if err := installChart(kubeconfigPath, chart.namespace, chart.name, chartPath); err != nil {
			return err
		}
	}

	return nil
}

// installChart is a helper function to install a single helm chart
func installChart(kubeconfigPath, namespace, chartName, chartPath string) error {
	client := kube.GetConfig(kubeconfigPath, "", namespace)
	actionConfig := &action.Configuration{}
	if err := actionConfig.Init(client, namespace, defaultHelmStorageDriver, log(namespace, chartName)); err != nil {
		util.UserOutput(fmt.Sprintf("Error initalizing helm: %v\n", err))
		return err
	}
	util.UserOutput(fmt.Sprintf("Loading chart %q\n", chartName))
	// load chart from the directory
	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}
	// set values to the default values of the chart
	values := chart.Values
	// values file is the same name as the chart name
	valuesFile := fmt.Sprintf("%s.yaml", chartPath)
	// if a valuesFile is provided use that as values
	valuesFileExists, err := isExists(valuesFile)
	if err != nil {
		// error checking for the existence values file
		return err
	}
	// load values file for a chart if values file exists
	if valuesFileExists {
		values, err = loadValuesFile(valuesFile)
		if err != nil {
			return err
		}
	}
	install := action.NewInstall(actionConfig)
	// Validate the chart and dependencies
	if err := validateChartAndDependencies(chart, install); err != nil {
		return err
	}
	install.ReleaseName = chartName
	install.Namespace = namespace
	install.CreateNamespace = true
	install.Wait = true
	release, err := install.Run(chart, values)
	if err != nil {
		return err
	}
	util.UserOutput(fmt.Sprintf("Release %q created\n", release.Name))

	return nil
}

// isExists true true or false if the file/directory is present or not
func isExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err

	}

	return true, nil
}

// validates the chart and its dependencies
func validateChartAndDependencies(chart *chart.Chart, client *action.Install) error {
	if err := chart.Validate(); err != nil {
		return err
	}
	// validate dependencies
	if req := chart.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chart, req); err != nil {
			if client.DependencyUpdate {
				man := &downloader.Manager{
					ChartPath:  chart.ChartFullPath(),
					Keyring:    client.ChartPathOptions.Keyring,
					SkipUpdate: false,
				}
				if err := man.Update(); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	return nil
}
