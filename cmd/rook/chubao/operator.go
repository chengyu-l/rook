/*
Copyright 2016 The Rook Authors. All rights reserved.

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
	"github.com/rook/rook/cmd/rook/rook"
	"github.com/rook/rook/pkg/operator/chubao"
	"github.com/rook/rook/pkg/util/flags"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

var operatorCmd = &cobra.Command{
	Use:   "operator",
	Short: "Runs the Chubao operator for orchestrating and managing Chubao storage in a Kubernetes cluster",
	Long: `Runs the Chubao operator for orchestrating and managing Chubao storage in a Kubernetes cluster
https://github.com/rook/rook`,
}

func init() {
	flags.SetFlagsFromEnv(operatorCmd.Flags(), rook.RookEnvVarPrefix)
	flags.SetLoggingFlags(operatorCmd.Flags())
	operatorCmd.RunE = startCluster
}

func startCluster(cmd *cobra.Command, args []string) error {
	rook.SetLogLevel()
	rook.LogStartupInfo(operatorCmd.Flags())

	//operatorNamespace := os.Getenv(k8sutil.PodNamespaceEnvVar)
	//if operatorNamespace == "" {
	//	rook.TerminateFatal(errors.Errorf("rook operator namespace is not provided. expose it via downward API in the rook operator manifest file using environment variable %q", k8sutil.PodNamespaceEnvVar))
	//}

	// Initialize signal handler
	signalChan := make(chan os.Signal, 1)
	stopChan := make(chan struct{})
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	// Start the controller-runtime Manager.
	mgrErrorChan := make(chan error)
	logger.Info("starting Rook Chubao manager")
	context := rook.NewContext()
	go chubao.StartManager(context, stopChan, mgrErrorChan)

	// Signal handler to stop the operator
	for {
		select {
		case <-stopChan:
			logger.Info("shutdown signal received, exiting...")
			return nil
		case err := <-mgrErrorChan:
			logger.Errorf("gave up to run the operator. %v", err)
			return err
		}
	}
}
