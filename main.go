package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ykoer/kube-allocated-resources/resources"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	// Define application options
	options := &resources.Options{}

	// TODO: Add long options
	flag.StringVar(&options.Labels, "l", "node-role.kubernetes.io/worker=", "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	flag.BoolVar(&options.GetTotalsByInstanceType, "g", false, "Return totals grouped by the instance type")
	flag.BoolVar(&options.GetNodeDetails, "d", false, "Return node details")
	outputFormat := flag.String("o", "json", "Output format. One of: json or yaml")

	flag.Parse()

	client := resources.NewAllocatedResourcesClient(createKubeClient(), options)
	clusterMetrics, err := client.GetAllocatedResources()

	if err != nil {
		panic(err)
	}

	var clusterMetricsOutput []byte
	if *outputFormat == "json" {
		clusterMetricsOutput, err = json.Marshal(clusterMetrics)
	} else if *outputFormat == "yaml" {
		clusterMetricsOutput, err = yaml.Marshal(clusterMetrics)
	}

	if err != nil {
		panic(err)
	}

	fmt.Println(string(clusterMetricsOutput))
}

func createKubeClient() *kubernetes.Clientset {
	config, err := getConfig()
	if err != nil {
		return nil
	}
	return kubernetes.NewForConfigOrDie(config)
}

func getConfig() (*rest.Config, error) {
	var config *rest.Config
	var err error

	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	if kubeconfig == "" {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	if err != nil {
		return nil, err
	}
	return config, nil
}
