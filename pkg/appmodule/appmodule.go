package appmodule

import (
	"context"
	"errors"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/uptrace/bun"
)

type Named interface {
	Name() string
}

type SchemaApplier interface {
	ApplySchema(context.Context, *bun.DB) error
}

type Registrar interface {
	Register(huma.API)
}

type Module interface {
	Named
	Registrar
}

type Closer interface {
	Close() error
}

type BuildFunc func(*Context) (Module, error)

type Spec struct {
	Name        string
	Requires    []string
	ApplySchema func(context.Context, *bun.DB) error
	Build       BuildFunc
}

type Context struct {
	ctx     context.Context
	db      *bun.DB
	modules map[string]Module
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) DB() *bun.DB {
	return c.db
}

func (c *Context) Get(name string) (Module, bool) {
	module, ok := c.modules[name]
	return module, ok
}

type Runtime struct {
	modules []Module
	byName  map[string]Module
}

func Build(ctx context.Context, db *bun.DB, specs ...Spec) (*Runtime, error) {
	ordered, err := sortSpecs(specs)
	if err != nil {
		return nil, err
	}
	buildCtx := &Context{
		ctx:     ctx,
		db:      db,
		modules: make(map[string]Module, len(ordered)),
	}
	runtime := &Runtime{
		modules: make([]Module, 0, len(ordered)),
		byName:  make(map[string]Module, len(ordered)),
	}
	for _, spec := range ordered {
		if err := applySchema(ctx, db, spec); err != nil {
			_ = runtime.Close()
			return nil, err
		}
		module, err := buildModule(buildCtx, spec)
		if err != nil {
			_ = runtime.Close()
			return nil, err
		}
		runtime.modules = append(runtime.modules, module)
		runtime.byName[module.Name()] = module
		buildCtx.modules[module.Name()] = module
	}
	return runtime, nil
}

func (r *Runtime) Register(api huma.API) {
	if r == nil {
		return
	}
	for _, module := range r.modules {
		module.Register(api)
	}
}

func (r *Runtime) Get(name string) (Module, bool) {
	if r == nil {
		return nil, false
	}
	module, ok := r.byName[name]
	return module, ok
}

func (r *Runtime) Close() error {
	if r == nil {
		return nil
	}
	var err error
	for i := len(r.modules) - 1; i >= 0; i-- {
		closer, ok := r.modules[i].(Closer)
		if !ok {
			continue
		}
		if closeErr := closer.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close %s module: %w", r.modules[i].Name(), closeErr))
		}
	}
	return err
}

func (r *Runtime) Modules() []Module {
	if r == nil {
		return nil
	}
	modules := make([]Module, len(r.modules))
	copy(modules, r.modules)
	return modules
}

func Get[T Module](r *Runtime, name string) (T, bool) {
	var zero T
	module, ok := r.Get(name)
	if !ok {
		return zero, false
	}
	value, ok := module.(T)
	if !ok {
		return zero, false
	}
	return value, true
}

func MustGet[T Module](r *Runtime, name string) T {
	module, ok := Get[T](r, name)
	if !ok {
		panic(fmt.Sprintf("appmodule: module %q not found or has unexpected type", name))
	}
	return module
}

func ContextGet[T Module](c *Context, name string) (T, bool) {
	var zero T
	module, ok := c.Get(name)
	if !ok {
		return zero, false
	}
	value, ok := module.(T)
	if !ok {
		return zero, false
	}
	return value, true
}

func MustContextGet[T Module](c *Context, name string) T {
	module, ok := ContextGet[T](c, name)
	if !ok {
		panic(fmt.Sprintf("appmodule: dependency %q not found or has unexpected type", name))
	}
	return module
}

func applySchema(ctx context.Context, db *bun.DB, spec Spec) error {
	if spec.ApplySchema == nil {
		return fmt.Errorf("module %s schema func is nil", spec.Name)
	}
	if err := spec.ApplySchema(ctx, db); err != nil {
		return fmt.Errorf("apply %s schema: %w", spec.Name, err)
	}
	return nil
}

func buildModule(ctx *Context, spec Spec) (Module, error) {
	if spec.Build == nil {
		return nil, fmt.Errorf("module %s build func is nil", spec.Name)
	}
	module, err := spec.Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("build %s module: %w", spec.Name, err)
	}
	if module == nil {
		return nil, fmt.Errorf("build %s module: nil module", spec.Name)
	}
	if module.Name() != spec.Name {
		return nil, fmt.Errorf("build %s module: returned module named %s", spec.Name, module.Name())
	}
	return module, nil
}

func sortSpecs(specs []Spec) ([]Spec, error) {
	byName := make(map[string]Spec, len(specs))
	for _, spec := range specs {
		if spec.Name == "" {
			return nil, errors.New("module name is empty")
		}
		if _, exists := byName[spec.Name]; exists {
			return nil, fmt.Errorf("duplicate module %s", spec.Name)
		}
		byName[spec.Name] = spec
	}

	const (
		unseen = iota
		visiting
		done
	)
	state := make(map[string]int, len(specs))
	var ordered []Spec
	var visit func(string, []string) error
	visit = func(name string, stack []string) error {
		switch state[name] {
		case done:
			return nil
		case visiting:
			return fmt.Errorf("module dependency cycle: %s -> %s", joinCycle(stack, name), name)
		}
		spec, ok := byName[name]
		if !ok {
			return fmt.Errorf("module dependency missing: %s", name)
		}
		state[name] = visiting
		stack = append(stack, name)
		for _, dep := range spec.Requires {
			if _, ok := byName[dep]; !ok {
				return fmt.Errorf("module %s requires missing module %s", name, dep)
			}
			if err := visit(dep, stack); err != nil {
				return err
			}
		}
		state[name] = done
		ordered = append(ordered, spec)
		return nil
	}
	for _, spec := range specs {
		if err := visit(spec.Name, nil); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

func joinCycle(stack []string, name string) string {
	for i, item := range stack {
		if item == name {
			stack = stack[i:]
			break
		}
	}
	result := ""
	for i, item := range stack {
		if i > 0 {
			result += " -> "
		}
		result += item
	}
	return result
}
