package cmd

import (
	"os"

	routeClient "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stakater/Xposer/internal/pkg/config"
	"github.com/stakater/Xposer/internal/pkg/constants"
	"github.com/stakater/Xposer/internal/pkg/controller"
	"github.com/stakater/Xposer/pkg/kube"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewXposerCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "xposer",
		Short: "A controller for watching services in your Kubernetes/Openshift cluster",
		Run:   startXposer,
	}
	return cmds
}

func startXposer(cmd *cobra.Command, args []string) {
	currentNamespace := os.Getenv("KUBERNETES_NAMESPACE")
	if len(currentNamespace) == 0 {
		currentNamespace = v1.NamespaceAll
		logrus.Warnf("Warning: KUBERNETES_NAMESPACE is unset, will monitor services in all namespaces.")
	}

	var kubeClient kubernetes.Interface
	var osClient *routeClient.RouteV1Client

	cfg, err := rest.InClusterConfig()
	if err != nil {
		kubeClient = kube.GetClientOutOfCluster()
	} else {
		kubeClient = kube.GetClient()
	}

	var clusterType = constants.KUBERNETES
	if kube.IsOpenShift(kubeClient.(*kubernetes.Clientset)) {
		clusterType = constants.OPENSHIFT
		osClient, err = routeClient.NewForConfig(cfg)
		if err != nil {
			logrus.Errorf("Can not create Openshift client with error: %v", err.Error())
		}
	}

	config := getControllerConfig()
	controller := controller.NewController(kubeClient, osClient, config, clusterType, currentNamespace)

	logrus.Infof("Controller started in the namespace: %v, with cluster type: %v", currentNamespace, clusterType)

	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(1, stop)

	// Wait forever
	select {}
}

func getClient() (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("HOME") + "/.kube/config"
	}
	//If kube config file exists in home so use that
	if _, err := os.Stat(kubeconfigPath); err == nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	} else {
		//use Incluster Configuration
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}
func getControllerConfig() config.Configuration {
	configFilePath := os.Getenv("CONFIG_FILE_PATH")
	if len(configFilePath) == 0 {
		configFilePath = "configs/config.yaml"
	}

	configuration, err := config.ReadConfig(configFilePath)
	if err != nil {
		logrus.Errorf("Can not read configuration file with the following error: %v", err)
	}
	return configuration
}
