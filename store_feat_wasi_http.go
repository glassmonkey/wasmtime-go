package wasmtime

// #include <wasmtime.h>
// #include <wasmtime/store.h>
import "C"
import "runtime"

// SetWasiHttp initializes the wasi-http context for this store.
//
// Must be called before instantiating a component that uses `wasi:http`.
// [Store.SetWasi] must have been called first to set up the underlying
// WASI Preview 2 context.
func (store *Store) SetWasiHttp() {
	C.wasmtime_context_set_wasi_http(store.Context())
	runtime.KeepAlive(store)
}
