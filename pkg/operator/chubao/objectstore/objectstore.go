package objectstore

import (
	"github.com/rook/rook/pkg/clusterd"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func Add(mgr manager.Manager, context *clusterd.Context) error {
	return nil
}

const (
	objectStoreQueueName = "chubao-monitor-queue"
)

type ObjectStoreEventHandler struct {
	cache.ResourceEventHandler

	context  *clusterd.Context
	queue    workqueue.RateLimitingInterface
	recorder record.EventRecorder
}

func New(context *clusterd.Context, recorder record.EventRecorder) *ObjectStoreEventHandler {
	return &ObjectStoreEventHandler{
		context:  context,
		recorder: recorder,
		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), objectStoreQueueName),
	}
}

// OnAdd calls AddFunc if it's not nil.
func (e *ObjectStoreEventHandler) OnAdd(obj interface{}) {
	//newCluster, ok := obj.(*cassandrav1alpha1.Cluster)
	//if !ok {
	//	return
	//}
	//cc.enqueueCluster(newCluster)
}

// OnUpdate calls UpdateFunc if it's not nil.
func (e *ObjectStoreEventHandler) OnUpdate(oldObj, newObj interface{}) {
	//newCluster, ok := newObj.(*cassandrav1alpha1.Cluster)
	//if !ok {
	//	return
	//}
	//oldCluster, ok := oldObj.(*cassandrav1alpha1.Cluster)
	//if !ok {
	//	return
	//}
	//// If the Spec is the same as the one in our cache, there aren't
	//// any changes we are interested in.
	//if reflect.DeepEqual(newCluster.Spec, oldCluster.Spec) {
	//	return
	//}
	//cc.enqueueCluster(newCluster)
}

// OnDelete calls DeleteFunc if it's not nil.
func (e *ObjectStoreEventHandler) OnDelete(obj interface{}) {

}

func (e *ObjectStoreEventHandler) RunWorker() {
	//fmt.Println("(cc *ObjectStoreEventHandler) RunWorker()")
}
