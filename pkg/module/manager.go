package module

import (
	"context"
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"github.com/go-logr/zapr"
	apiv1 "github.com/stefanprodan/timoni/api/v1alpha1"
	"github.com/stefanprodan/timoni/internal/engine"
	"github.com/stefanprodan/timoni/internal/engine/fetcher"
	"github.com/stefanprodan/timoni/internal/reconciler"
	"go.uber.org/zap"
	"os"
	"time"
)

type Manager struct {
	Name        string
	Source      string
	Version     string
	Namespace   string
	Credentials string
	Module      *apiv1.ModuleReference
	Builder     *engine.ModuleBuilder
	CueCtx      *cue.Context
	Ctx         context.Context
	Values      map[string]interface{}
	Rcg         *RESTClientGetter
	TempDir     string
	CacheDir    string
	ModuleRoot  string
}

func NewManager(ctx context.Context, name string, source string, version string, namespace string, credentials string, values map[string]interface{}) (*Manager, error) {
	rcg, err := NewRESTClientGetter()
	if err != nil {
		return nil, err
	}
	return &Manager{
		Name:        name,
		Source:      source,
		Version:     version,
		Namespace:   namespace,
		Credentials: credentials,
		CueCtx:      cuecontext.New(),
		Ctx:         ctx,
		Values:      values,
		Rcg:         rcg,
		CacheDir:    "./.timoni/cache",
	}, nil
}

func (m *Manager) fetch() (*apiv1.ModuleReference, error) {
	tmpDir, err := os.MkdirTemp("", apiv1.FieldManager)
	if err != nil {
		return nil, err
	}
	f, err := fetcher.New(context.Background(), fetcher.Options{
		Source:       m.Source,
		Version:      m.Version,
		Destination:  tmpDir,
		CacheDir:     m.CacheDir,
		Creds:        m.Credentials,
		Insecure:     false,
		DefaultLocal: false,
	})
	if err != nil {
		return nil, err
	}
	m.TempDir = tmpDir
	m.ModuleRoot = f.GetModuleRoot()
	return f.Fetch()
}

func (m *Manager) Build() (cue.Value, error) {
	mod, err := m.fetch()
	m.Module = mod
	if err != nil {
		return cue.Value{}, err
	}
	m.Builder = engine.NewModuleBuilder(
		m.CueCtx,
		m.Name,
		m.Namespace,
		m.ModuleRoot,
		"main",
	)

	encoded := m.CueCtx.Encode(m.Values)
	syn := encoded.Syntax()
	bs, err := format.Node(syn)
	if err != nil {
		return cue.Value{}, err
	}
	bsArray := make([][]byte, 1)
	bsArray[0] = bs

	if err = m.Builder.WriteSchemaFile(); err != nil {
		return cue.Value{}, err
	}
	m.Module.Name, err = m.Builder.GetModuleName()
	if err != nil {
		return cue.Value{}, err
	}
	err = m.Builder.MergeValuesFile(bsArray)
	if err != nil {
		return cue.Value{}, err
	}
	return m.Builder.Build()
}

func (m *Manager) Apply() error {
	buildResult, err := m.Build()
	if err != nil {
		return err
	}
	instance := &engine.BundleInstance{
		Name:      m.Name,
		Namespace: m.Namespace,
		Module:    *m.Module,
		Bundle:    "",
	}
	zapLog, _ := zap.NewDevelopment() // or NewProduction, or New(zapcore.Config)
	log := zapr.NewLogger(zapLog)
	r := reconciler.NewReconciler(log,
		&reconciler.CommonOptions{
			Dir:                m.TempDir,
			Wait:               false,
			Force:              false,
			OverwriteOwnership: false,
		},
		5*time.Minute,
	)
	kubeconfigArgs, err := NewRESTClientGetter()
	if err != nil {
		return err
	}
	if err = r.Init(m.Ctx, m.Builder, buildResult, instance, kubeconfigArgs); err != nil {
		return err
	}
	return r.ApplyInstance(m.Ctx, log, m.Builder, buildResult)
}
