package bundler

import (
	"context"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"github.com/go-logr/zapr"
	apiv1 "github.com/stefanprodan/timoni/api/v1alpha1"
	"github.com/stefanprodan/timoni/internal/engine"
	"github.com/stefanprodan/timoni/internal/engine/fetcher"
	"github.com/stefanprodan/timoni/internal/logger"
	"github.com/stefanprodan/timoni/internal/reconciler"
	"go.uber.org/zap"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
	"time"
)

func main() {
	tmpDir, err := os.MkdirTemp("", apiv1.FieldManager)
	if err != nil {
		panic(err)
	}
	f, err := fetcher.New(context.Background(), fetcher.Options{
		Source:      "oci://ghcr.io/stefanprodan/modules/podinfo",
		Version:     "latest",
		Destination: tmpDir,
		CacheDir:    "/Users/edricardo/.timoni/cache",
		Creds:       "",
		Insecure:    false,
		// DefaultLocal: false,
	})
	if err != nil {
		panic(err)
	}
	mod, err := f.Fetch()
	if err != nil {
		panic(err)
	}
	builder := engine.NewModuleBuilder(
		cuecontext.New(),
		"podinfo",
		"test",
		f.GetModuleRoot(),
		"main",
	)
	_ = builder
	_ = mod

	ctx := cuecontext.New()

	vals := map[string]interface{}{
		"values": map[string]interface{}{
			"resources": map[string]interface{}{
				"requests": map[string]interface{}{
					"cpu":    "100m",
					"memory": "100Mi",
				},
				"limits": map[string]interface{}{
					"cpu":    "100m",
					"memory": "100Mi",
				},
			},
		},
	}

	encoded := ctx.Encode(vals)
	syn := encoded.Syntax()
	bs, err := format.Node(syn)
	if err != nil {
		panic(err)
	}
	bsArray := make([][]byte, 1)
	bsArray[0] = bs

	if err = builder.WriteSchemaFile(); err != nil {
		panic(err)
	}
	mod.Name, err = builder.GetModuleName()
	if err != nil {
		panic(err)
	}
	err = builder.MergeValuesFile(bsArray)
	if err != nil {
		panic(err)
	}

	buildResult, err := builder.Build()
	if err != nil {
		panic(err)
	}

	//applySets, err := builder.GetApplySets(buildResult)
	//if err != nil {
	//	panic(err)
	//}

	//var objects []*unstructured.Unstructured
	//for _, set := range applySets {
	//	objects = append(objects, set.Objects...)
	//}

	ctxNew := context.Background()

	instance := &engine.BundleInstance{
		Name:      "podinfo",
		Namespace: "test",
		Module:    *mod,
		Bundle:    "",
	}
	zapLog, _ := zap.NewDevelopment() // or NewProduction, or New(zapcore.Config)
	log := zapr.NewLogger(zapLog)
	r := reconciler.NewInteractiveReconciler(log,
		&reconciler.CommonOptions{
			Dir:                tmpDir,
			Wait:               false,
			Force:              false,
			OverwriteOwnership: false,
		},
		&reconciler.InteractiveOptions{
			DryRun:        false,
			Diff:          false,
			DiffOutput:    os.Stdout,
			ProgressStart: logger.StartSpinner,
		},
		5*time.Minute,
	)
	kubeconfigArgs := genericclioptions.NewConfigFlags(false)
	if err := r.Init(ctxNew, builder, buildResult, instance, kubeconfigArgs); err != nil {
		panic(err)
	}
	err = r.ApplyInstance(ctxNew, log,
		builder,
		buildResult,
	)
	if err != nil {
		panic(err)
	}

}
