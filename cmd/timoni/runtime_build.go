/*
Copyright 2023 Stefan Prodan

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
	"context"
	"errors"
	"fmt"
	"os"
	"sort"

	"cuelang.org/go/cue/cuecontext"
	"github.com/spf13/cobra"

	apiv1 "github.com/stefanprodan/timoni/api/v1alpha1"
	"github.com/stefanprodan/timoni/internal/engine"
	"github.com/stefanprodan/timoni/internal/logger"
	"github.com/stefanprodan/timoni/internal/runtime"
)

var runtimeBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build validates the runtime definition, queries the cluster, extracts the values and prints them",
	Example: `  #  Print the runtime values from a cluster
  timoni runtime build -f runtime.cue
`,
	Args: cobra.NoArgs,
	RunE: runRuntimeBuildCmd,
}

type runtimeBuildFlags struct {
	files                []string
	clusterSelector      string
	clusterGroupSelector string
}

var runtimeBuildArgs runtimeBuildFlags

func init() {
	runtimeBuildCmd.Flags().StringSliceVarP(&runtimeBuildArgs.files, "file", "f", nil,
		"The local path to runtime.cue files.")
	runtimeBuildCmd.Flags().StringVar(&runtimeBuildArgs.clusterSelector, "cluster", "*",
		"Select cluster by name.")
	runtimeBuildCmd.Flags().StringVar(&runtimeBuildArgs.clusterGroupSelector, "cluster-group", "*",
		"Select clusters by group name.")
	runtimeCmd.AddCommand(runtimeBuildCmd)
}

func runRuntimeBuildCmd(cmd *cobra.Command, args []string) error {
	files := runtimeBuildArgs.files
	if len(files) == 0 {
		return errors.New("no runtime provided with -f")
	}
	var stdinFile string
	for i, file := range files {
		if file == "-" {
			stdinFile, err := saveReaderToFile(cmd.InOrStdin())
			if err != nil {
				return err
			}
			files[i] = stdinFile
			break
		}
	}
	if stdinFile != "" {
		defer os.Remove(stdinFile)
	}

	rt, err := buildRuntime(files)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), rootArgs.timeout)
	defer cancel()

	clusters := rt.SelectClusters(runtimeBuildArgs.clusterSelector, runtimeBuildArgs.clusterGroupSelector)
	if len(clusters) == 0 {
		return errors.New("no cluster found")
	}

	for _, cluster := range clusters {
		log := loggerRuntime(cmd.Context(), rt.Name, cluster.Name, true)

		kubeconfigArgs.Context = &cluster.KubeContext
		rm, err := runtime.NewResourceManager(kubeconfigArgs)
		if err != nil {
			return err
		}

		reader := runtime.NewResourceReader(rm)

		values, err := reader.Read(ctx, rt.Refs)
		if err != nil {
			return err
		}

		keys := make([]string, 0, len(values))

		for k := range values {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			log.Info(fmt.Sprintf("%s: %s", logger.ColorizeSubject(k), values[k]))
		}

		if len(values) == 0 {
			log.Info("no values defined")
		}
	}

	return nil
}

func buildRuntime(files []string) (*apiv1.Runtime, error) {
	defaultRuntime := apiv1.DefaultRuntime(*kubeconfigArgs.Context)
	if len(files) == 0 {
		return defaultRuntime, nil
	}

	tmpDir, err := os.MkdirTemp("", apiv1.FieldManager)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	ctx := cuecontext.New()
	rb := engine.NewRuntimeBuilder(ctx, files)

	if err := rb.InitWorkspace(tmpDir); err != nil {
		return nil, describeErr(tmpDir, "failed to init runtime", err)
	}

	v, err := rb.Build(tmpDir)
	if err != nil {
		return nil, describeErr(tmpDir, "failed to parse runtime", err)
	}

	rt, err := rb.GetRuntime(v)
	if err != nil {
		return nil, err
	}

	if len(rt.Clusters) == 0 {
		rt.Clusters = defaultRuntime.Clusters
	}
	return rt, nil
}
