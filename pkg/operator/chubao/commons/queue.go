package commons

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func EnqueueItem(queue workqueue.RateLimitingInterface, obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	queue.AddRateLimited(key)
}

func ProcessNextWorkItem(queue workqueue.RateLimitingInterface, workFunc func(key string) error) bool {
	obj, shutdown := queue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer queue.Done(obj)
		key, ok := obj.(string)
		if !ok {
			queue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in queue but got %#v", obj))
		}
		if err := workFunc(key); err != nil {
			queue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s', requeueing: %s", key, err.Error())
		}
		queue.Forget(obj)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}
