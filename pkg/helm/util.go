package helm

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"sigs.k8s.io/yaml"

	"github.com/kubernetes-sigs/bootkube/pkg/util"
)

type chartInfo struct {
	name      string
	namespace string
}

// getCharts returns the map structure of charts found in sub directories
// Sub directories of chartsDir corresponds to the namespaces the charts are to be installed.
// Each namespace sub directory contains the respectie charts.
// This method returns the map structure of namespace directory name as key and path of the charts as values.
func getCharts(chartsDir string) ([]chartInfo, error) {
	charts := []chartInfo{}
	namespaces, err := getDirs(chartsDir)
	if err != nil {
		return nil, err
	}

	for _, namespace := range namespaces {
		ch, err := getDirs(filepath.Join(chartsDir, namespace))
		if err != nil {
			return nil, err
		}

		for _, c := range ch {
			charts = append(charts, chartInfo{
				namespace: namespace,
				name:      c,
			})
		}
	}

	return charts, nil
}

// getDirs returns the list of directories in the given path.
func getDirs(path string) ([]string, error) {
	var dirs = []string{}
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return []string{}, err
	}

	for _, file := range files {
		if file.IsDir() {
			dirs = append(dirs, file.Name())
		}
	}

	return dirs, nil
}

// loadValuesFile reads the file from given path and returns the content in
// a map of key value pairs.
func loadValuesFile(path string) (map[string]interface{}, error) {
	values := map[string]interface{}{}
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(bytes, &values); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %v", path, err)
	}

	return values, nil
}

// log is a wrapper over UserOutput function to be passed to Helm client,
// as Helm logs comes without newline at the end, which makes the output
// very difficult to read. This function simply appends the newline at the end
// of a given format.
func log(namespace, chartName string) func(format string, a ...interface{}) {
	return func(format string, a ...interface{}) {
		util.UserOutput(fmt.Sprintf("%s/%s: %s\n", namespace, chartName, format), a)
	}
}
