package wasmtime

// #include <wasmtime.h>
// #include <wasmtime/component/component.h>
import "C"
import (
	"runtime"
	"unsafe"
)

// Component is a compiled WebAssembly component.
//
// A Component is produced from a component-model wasm binary (e.g. the output
// of `cargo component build` or `wasm-tools component new`). It is roughly the
// component-model equivalent of [Module].
//
// The owning [Engine] must have been configured with
// [Config.SetWasmComponentModel] enabled.
type Component struct {
	_ptr *C.wasmtime_component_t
}

// NewComponent compiles a [Component] from the given wasm component bytes.
//
// The provided bytes must be a wasm component (preamble starting with
// `\0asm\x0d\0\x01\0`). For a core wasm module use [NewModule] instead.
func NewComponent(engine *Engine, wasm []byte) (*Component, error) {
	var wasmPtr *C.uint8_t
	if len(wasm) > 0 {
		wasmPtr = (*C.uint8_t)(unsafe.Pointer(&wasm[0]))
	}
	var ptr *C.wasmtime_component_t
	err := C.wasmtime_component_new(engine.ptr(), wasmPtr, C.size_t(len(wasm)), &ptr)
	runtime.KeepAlive(engine)
	runtime.KeepAlive(wasm)
	if err != nil {
		return nil, mkError(err)
	}
	return mkComponent(ptr), nil
}

// NewComponentDeserialize builds a [Component] from a previously serialized
// component (see [Component.Serialize]).
//
// This function is **not safe to call with untrusted input**. The bytes must
// have been produced by a matching wasmtime version using
// [Component.Serialize].
func NewComponentDeserialize(engine *Engine, serialized []byte) (*Component, error) {
	var bufPtr *C.uint8_t
	if len(serialized) > 0 {
		bufPtr = (*C.uint8_t)(unsafe.Pointer(&serialized[0]))
	}
	var ptr *C.wasmtime_component_t
	err := C.wasmtime_component_deserialize(engine.ptr(), bufPtr, C.size_t(len(serialized)), &ptr)
	runtime.KeepAlive(engine)
	runtime.KeepAlive(serialized)
	if err != nil {
		return nil, mkError(err)
	}
	return mkComponent(ptr), nil
}

func mkComponent(ptr *C.wasmtime_component_t) *Component {
	c := &Component{_ptr: ptr}
	runtime.SetFinalizer(c, func(c *Component) {
		c.Close()
	})
	return c
}

func (c *Component) ptr() *C.wasmtime_component_t {
	ret := c._ptr
	if ret == nil {
		panic("component has been closed already")
	}
	maybeGC()
	return ret
}

// Close deallocates this component's resources explicitly.
//
// After calling Close any further use of the component will panic. The
// finalizer set on construction will also call this so explicit Close is
// optional but recommended for deterministic cleanup.
func (c *Component) Close() {
	if c._ptr == nil {
		return
	}
	runtime.SetFinalizer(c, nil)
	C.wasmtime_component_delete(c._ptr)
	c._ptr = nil
}

// Serialize converts this in-memory compiled component into a byte vector that
// can be later deserialized via [NewComponentDeserialize].
func (c *Component) Serialize() ([]byte, error) {
	retVec := C.wasm_byte_vec_t{}
	err := C.wasmtime_component_serialize(c.ptr(), &retVec)
	runtime.KeepAlive(c)
	if err != nil {
		return nil, mkError(err)
	}
	out := C.GoBytes(unsafe.Pointer(retVec.data), C.int(retVec.size))
	C.wasm_byte_vec_delete(&retVec)
	return out, nil
}
