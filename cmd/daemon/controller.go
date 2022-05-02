/*
Copyright 2022.

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

package main

import (
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var (
	controllerCmd = &cobra.Command{
		Use:   "controller",
		Short: "Manages local Windows Services",
		Long: "Manages the state of Windows Services, according to information given by Windows Service ConfigMaps " +
			"present within the cluster",
		Run: runControllerCmd,
	}
)

func init() {
	rootCmd.AddCommand(controllerCmd)
}

func runControllerCmd(cmd *cobra.Command, args []string) {
	klog.Info("to be implemented")
}
