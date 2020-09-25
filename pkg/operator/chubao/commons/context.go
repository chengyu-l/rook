package commons

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Context struct {
	Client    client.Client
	Cache     cache.Cache
	ClientSet kubernetes.Interface
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
}
