package wasmtime

// #include <wasmtime.h>
// #include <wasmtime/component/linker.h>
import "C"
import "runtime"

// ComponentLinker links host-provided imports for a [Component] and
// instantiates it within a [Store].
//
// This is the component-model equivalent of [Linker]. WASI Preview 2 and
// wasi-http imports can be added with [ComponentLinker.DefineWasi] and
// [ComponentLinker.DefineWasiHttp].
type ComponentLinker struct {
	_ptr   *C.wasmtime_component_linker_t
	Engine *Engine
}

// NewComponentLinker creates a fresh [ComponentLinker] bound to the given
// engine.
//
// The engine must have been configured with [Config.SetWasmComponentModel]
// enabled.
func NewComponentLinker(engine *Engine) *ComponentLinker {
	ptr := C.wasmtime_component_linker_new(engine.ptr())
	runtime.KeepAlive(engine)
	l := &ComponentLinker{_ptr: ptr, Engine: engine}
	runtime.SetFinalizer(l, func(l *ComponentLinker) {
		l.Close()
	})
	return l
}

func (l *ComponentLinker) ptr() *C.wasmtime_component_linker_t {
	ret := l._ptr
	if ret == nil {
		panic("component linker has been closed already")
	}
	maybeGC()
	return ret
}

// Close deallocates this linker's resources explicitly.
func (l *ComponentLinker) Close() {
	if l._ptr == nil {
		return
	}
	runtime.SetFinalizer(l, nil)
	C.wasmtime_component_linker_delete(l._ptr)
	l._ptr = nil
}

// AllowShadowing configures whether names can be redefined after they've
// already been defined in this linker.
func (l *ComponentLinker) AllowShadowing(allow bool) {
	C.wasmtime_component_linker_allow_shadowing(l.ptr(), C.bool(allow))
	runtime.KeepAlive(l)
}

// DefineWasi adds the entire WASI Preview 2 surface (wasi:cli, wasi:clocks,
// wasi:filesystem, wasi:io, wasi:random, wasi:sockets) to this linker.
//
// The store passed to [ComponentLinker.Instantiate] must have a WASI config
// set via [Store.SetWasi] for these imports to actually do anything at
// runtime.
func (l *ComponentLinker) DefineWasi() error {
	err := C.wasmtime_component_linker_add_wasip2(l.ptr())
	runtime.KeepAlive(l)
	if err != nil {
		return mkError(err)
	}
	return nil
}

// DefineWasiHttp adds the wasi:http/types and wasi:http/outgoing-handler
// interfaces to this linker. [ComponentLinker.DefineWasi] must be called
// first to provide the underlying WASI Preview 2 dependencies.
func (l *ComponentLinker) DefineWasiHttp() error {
	err := C.wasmtime_component_linker_add_wasi_http(l.ptr())
	runtime.KeepAlive(l)
	if err != nil {
		return mkError(err)
	}
	return nil
}

// DefineUnknownImportsAsTraps marks any unresolved imports of the given
// component as trapping functions. Useful for getting a component to
// instantiate even when not all of its imports have host implementations.
func (l *ComponentLinker) DefineUnknownImportsAsTraps(component *Component) error {
	err := C.wasmtime_component_linker_define_unknown_imports_as_traps(l.ptr(), component.ptr())
	runtime.KeepAlive(l)
	runtime.KeepAlive(component)
	if err != nil {
		return mkError(err)
	}
	return nil
}

// Instantiate creates a [ComponentInstance] of the given component within the
// given store, satisfying the component's imports from this linker.
func (l *ComponentLinker) Instantiate(store Storelike, component *Component) (*ComponentInstance, error) {
	var inst C.wasmtime_component_instance_t
	err := C.wasmtime_component_linker_instantiate(
		l.ptr(),
		store.Context(),
		component.ptr(),
		&inst,
	)
	runtime.KeepAlive(l)
	runtime.KeepAlive(component)
	runtime.KeepAlive(store)
	if err != nil {
		return nil, mkError(err)
	}
	return &ComponentInstance{raw: inst}, nil
}
