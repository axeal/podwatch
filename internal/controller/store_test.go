package controller

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"
)

func TestStore(t *testing.T) {

	clientSet := fake.NewSimpleClientset()

	//should return no events for Pods in other Namespace

	t.Run("should return one add event for one existing Pod", func(t *testing.T) {
		ns := createNamespace(clientSet, t)
		defer deleteNamespace(ns, clientSet, t)

		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod",
				Namespace: ns,
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  "busybox",
						Image: "gcr.io/google_containers/echoserver:1.4",
					},
				},
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
			},
		}

		createPod(pod, ns, clientSet, t)

		stopCh := make(chan struct{})
		defer close(stopCh)

		var add uint64
		var upd uint64
		var del uint64

		store := NewStore(clientSet, ns)
		store.Run(stopCh)

		go processQueue(store.queue, &add, &upd, &del)
		time.Sleep(time.Second)

		if atomic.LoadUint64(&add) != 1 {
			t.Errorf("expected 1 events of type Create but %v occurred", add)
		}
		if atomic.LoadUint64(&upd) != 0 {
			t.Errorf("expected 0 events of type Update but %v occurred", upd)
		}
		if atomic.LoadUint64(&del) != 0 {
			t.Errorf("expected 0 events of type Delete but %v occurred", del)
		}

	})
	//should return one add event for one existing Pod
	//should return two events for add of one existing and one new Pod, one update of Pod, and two delete events
}

func processQueue(queue workqueue.Interface, add *uint64, upd *uint64, del *uint64) {
	for {
		e, term := queue.Get()
		if term {
			return
		}
		switch e.(Event).Type {
		case CreateEvent:
			atomic.AddUint64(add, 1)
		case UpdateEvent:
			atomic.AddUint64(upd, 1)
		case DeleteEvent:
			atomic.AddUint64(del, 1)
		}
		queue.Done(e)
	}
}

func createPod(pod *v1.Pod, namespace string, clientSet kubernetes.Interface, t *testing.T) *v1.Pod {
	t.Helper()

	p, err := clientSet.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating Pod: %v", err)
	}

	return p
}

//https://github.com/kubernetes/ingress-nginx/blob/5a5bff1fb98c896192ca58ecf51fa5a8985d2282/internal/ingress/controller/store/store_test.go#L1227
func createNamespace(clientSet kubernetes.Interface, t *testing.T) string {
	t.Helper()

	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("store-test-%v", time.Now().Unix()),
		},
	}

	ns, err := clientSet.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	if err != nil {
		t.Errorf("Error creating the namespace: %v", err)
	}

	return ns.Name
}

//https://github.com/kubernetes/ingress-nginx/blob/5a5bff1fb98c896192ca58ecf51fa5a8985d2282/internal/ingress/controller/store/store_test.go#L1244
func deleteNamespace(ns string, clientSet kubernetes.Interface, t *testing.T) {
	t.Helper()

	err := clientSet.CoreV1().Namespaces().Delete(context.TODO(), ns, metav1.DeleteOptions{})
	if err != nil {
		t.Errorf("Error deleting the namespace: %v", err)
	}
}
