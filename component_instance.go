package wasmtime

// #include <wasmtime.h>
// #include <wasmtime/component/instance.h>
import "C"
import (
	"fmt"
	"runtime"
	"unsafe"
)

// ComponentInstance is an instantiated [Component] within a [Store].
//
// Like [Instance] for core wasm, ComponentInstance is a value-type handle
// scoped to a particular store. Passing it to a different store will trigger
// an assertion in wasmtime.
type ComponentInstance struct {
	raw C.wasmtime_component_instance_t
}

// GetFunc looks up an exported component function by name. Returns nil if no
// function with the given name is exported.
func (i *ComponentInstance) GetFunc(store Storelike, name string) *ComponentFunc {
	var nameBytes *C.char
	var nameLen C.size_t
	nameBuf := []byte(name)
	if len(nameBuf) > 0 {
		nameBytes = (*C.char)(unsafe.Pointer(&nameBuf[0]))
		nameLen = C.size_t(len(nameBuf))
	}

	idx := C.wasmtime_component_instance_get_export_index(
		&i.raw,
		store.Context(),
		nil,
		nameBytes,
		nameLen,
	)
	runtime.KeepAlive(nameBuf)
	runtime.KeepAlive(name)
	runtime.KeepAlive(store)
	if idx == nil {
		return nil
	}
	defer C.wasmtime_component_export_index_delete(idx)

	var fn C.wasmtime_component_func_t
	ok := C.wasmtime_component_instance_get_func(
		&i.raw,
		store.Context(),
		idx,
		&fn,
	)
	runtime.KeepAlive(store)
	if !bool(ok) {
		return nil
	}
	return &ComponentFunc{raw: fn}
}

// MustGetFunc is like [ComponentInstance.GetFunc] but panics if the named
// function is not exported.
func (i *ComponentInstance) MustGetFunc(store Storelike, name string) *ComponentFunc {
	fn := i.GetFunc(store, name)
	if fn == nil {
		panic(fmt.Sprintf("component does not export function %q", name))
	}
	return fn
}
