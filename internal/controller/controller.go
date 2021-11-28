package controller

import (
	"path/filepath"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Controller struct {
	store  Store
	stopCh chan struct{}
}

func NewController() (*Controller, error) {
	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logrus.Fatalf("Error reading kubeconfig: %v", err)
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logrus.Fatalf("Error initialising Kubernetes client: %v", err)
		return nil, err
	}

	controller := &Controller{
		store:  NewStore(clientset, "default"),
		stopCh: make(chan struct{}),
	}
	return controller, nil
}

func (c *Controller) Start() {
	logrus.Info("Starting the podwatch controller")

	c.store.Run(c.stopCh)
}

func (c *Controller) Stop() {
	close(c.stopCh)
}
