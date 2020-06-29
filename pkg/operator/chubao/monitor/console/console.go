package console

import (
	"github.com/rook/rook/pkg/clusterd"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func Add(mgr manager.Manager, context *clusterd.Context) error {
	return nil
}
