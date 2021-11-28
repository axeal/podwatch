package controller

import (
	"fmt"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Store struct {
	informer cache.SharedIndexInformer
	queue    workqueue.RateLimitingInterface
	store    cache.Store
}

// https://github.com/kubernetes/ingress-nginx/blob/5a5bff1fb98c896192ca58ecf51fa5a8985d2282/internal/ingress/controller/store/store.go#L103
type EventType string

const (
	CreateEvent EventType = "CREATE"
	UpdateEvent EventType = "UPDATE"
	DeleteEvent EventType = "DELETE"
)

type Event struct {
	Key  string
	Type EventType
}

func NewStore(client kubernetes.Interface, ns string) Store {
	q := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	listOptions := informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
		opts.FieldSelector = "status.phase=Running"
	})
	factory := informers.NewSharedInformerFactoryWithOptions(client, 0, informers.WithNamespace(ns), listOptions)
	podInformer := factory.Core().V1().Pods().Informer()

	nodeEventHandlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				logrus.Errorf("MetaNamespaceKeyFunc failed to get key for Pod (%v): %v", obj, err)
			} else {
				event := Event{
					Key:  key,
					Type: CreateEvent,
				}
				q.Add(event)
				logrus.Infof("Event received of type [%s] for [%s]", event.Type, event.Key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err != nil {
				logrus.Errorf("MetaNamespaceKeyFunc failed to get key for Pod (%v): %v", obj, err)
			} else {
				event := Event{
					Key:  key,
					Type: DeleteEvent,
				}
				q.Add(event)
				logrus.Infof("Event received of type [%s] for [%s]", event.Type, event.Key)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(old)
			if err != nil {
				logrus.Errorf("MetaNamespaceKeyFunc failed to get key for Pod (%v): %v", old, err)
			} else {
				event := Event{
					Key:  key,
					Type: UpdateEvent,
				}
				q.Add(event)
				logrus.Infof("Event received of type [%s] for [%s]", event.Type, event.Key)
			}
		},
	}

	podInformer.AddEventHandler(nodeEventHandlers)

	return Store{
		informer: podInformer,
		queue:    q,
		store:    podInformer.GetStore(),
	}
}

func (s *Store) Run(stopCh chan struct{}) {
	go s.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, s.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
	}

}
