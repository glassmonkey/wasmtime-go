package wasmtime

// #include <stdlib.h>
// #include <wasmtime.h>
// #include <wasmtime/component/func.h>
// #include <wasmtime/component/val.h>
import "C"
import (
	"runtime"
	"unsafe"
)

// ComponentFunc is an exported function from a [ComponentInstance].
//
// Like [Func] for core wasm, ComponentFunc is a value-type handle scoped to a
// particular store.
type ComponentFunc struct {
	raw C.wasmtime_component_func_t
}

// Call invokes this component function with the given args and returns the
// results.
//
// The number of args and result slots must match the function's actual
// signature (this binding does not yet expose the type, so callers must
// know the WIT signature out-of-band).
func (f *ComponentFunc) Call(store Storelike, args []ComponentVal, numResults int) ([]ComponentVal, error) {
	// Marshal args.
	var argPtr *C.wasmtime_component_val_t
	if len(args) > 0 {
		argPtr = (*C.wasmtime_component_val_t)(C.calloc(C.size_t(len(args)), C.size_t(unsafe.Sizeof(C.wasmtime_component_val_t{}))))
		if argPtr == nil {
			panic("calloc failed")
		}
		defer func() {
			for i := 0; i < len(args); i++ {
				entry := (*C.wasmtime_component_val_t)(unsafe.Pointer(uintptr(unsafe.Pointer(argPtr)) + uintptr(i)*unsafe.Sizeof(C.wasmtime_component_val_t{})))
				C.wasmtime_component_val_delete(entry)
			}
			C.free(unsafe.Pointer(argPtr))
		}()
		for i, a := range args {
			entry := (*C.wasmtime_component_val_t)(unsafe.Pointer(uintptr(unsafe.Pointer(argPtr)) + uintptr(i)*unsafe.Sizeof(C.wasmtime_component_val_t{})))
			if err := a.toCVal(entry); err != nil {
				return nil, err
			}
		}
	}

	// Allocate result slots (zero-initialised).
	var resPtr *C.wasmtime_component_val_t
	if numResults > 0 {
		resPtr = (*C.wasmtime_component_val_t)(C.calloc(C.size_t(numResults), C.size_t(unsafe.Sizeof(C.wasmtime_component_val_t{}))))
		if resPtr == nil {
			panic("calloc failed")
		}
		defer func() {
			for i := 0; i < numResults; i++ {
				entry := (*C.wasmtime_component_val_t)(unsafe.Pointer(uintptr(unsafe.Pointer(resPtr)) + uintptr(i)*unsafe.Sizeof(C.wasmtime_component_val_t{})))
				C.wasmtime_component_val_delete(entry)
			}
			C.free(unsafe.Pointer(resPtr))
		}()
	}

	cerr := C.wasmtime_component_func_call(
		&f.raw,
		store.Context(),
		argPtr,
		C.size_t(len(args)),
		resPtr,
		C.size_t(numResults),
	)
	runtime.KeepAlive(store)
	if cerr != nil {
		return nil, mkError(cerr)
	}

	// Read results back into Go.
	out := make([]ComponentVal, numResults)
	for i := 0; i < numResults; i++ {
		entry := (*C.wasmtime_component_val_t)(unsafe.Pointer(uintptr(unsafe.Pointer(resPtr)) + uintptr(i)*unsafe.Sizeof(C.wasmtime_component_val_t{})))
		v, err := fromCVal(entry)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}
