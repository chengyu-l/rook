/*
Copyright 2019 The Rook Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package chubao

import (
	"github.com/coreos/pkg/capnslog"
	"github.com/pkg/errors"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/clusterd"
	"github.com/rook/rook/pkg/operator/chubao/cluster"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var logger = capnslog.NewPackageLogger("github.com/rook/rook", "rook-chubao-operator")

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs = []func(manager.Manager, commons.Context) error{
	cluster.Add,
}

func StartManager(context *clusterd.Context, stopCh <-chan struct{}, mgrErrorCh chan error) {
	// Set up a manager
	mgrOpts := manager.Options{
		LeaderElection: false,
		Namespace:      v1.NamespaceAll,
	}

	logger.Info("setting up the controller-runtime manager")
	kubeConfig, err := config.GetConfig()
	if err != nil {
		mgrErrorCh <- errors.Wrap(err, "failed to get client config for controller-runtime manager")
		return
	}

	mgr, err := manager.New(kubeConfig, mgrOpts)
	if err != nil {
		mgrErrorCh <- errors.Wrap(err, "failed to set up overall controller-runtime manager")
		return
	}

	// Add the registered scheme to the manager
	err = chubaoapi.AddToScheme(mgr.GetScheme())
	if err != nil {
		mgrErrorCh <- errors.Wrap(err, "failed to add scheme to controller-runtime manager")
		return
	}

	// Add the registered controllers to the manager
	err = AddToManager(mgr, context)
	if err != nil {
		mgrErrorCh <- errors.Wrap(err, "failed to add controllers to controller-runtime manager")
		return
	}

	logger.Info("starting the controller-runtime manager")
	if err := mgr.Start(stopCh); err != nil {
		mgrErrorCh <- errors.Wrap(err, "unable to run the controller-runtime manager")
		return
	}
}

func AddToManager(mgr manager.Manager, context *clusterd.Context) error {
	cxt := commons.Context{
		Client:    mgr.GetClient(),
		ClientSet: context.Clientset,
		Recorder:  mgr.GetEventRecorderFor("rook-chubao-controller"),
		Scheme:    mgr.GetScheme(),
	}

	for _, f := range AddToManagerFuncs {
		if err := f(mgr, cxt); err != nil {
			return err
		}
	}

	return nil
}
