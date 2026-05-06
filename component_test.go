package wasmtime

import (
	"os"
	"testing"
)

// htmlFetcherWasmPath returns the path to a Rust-built wasi-http
// `html_fetcher` component used for the wasi-http smoke test, or empty if
// unavailable.
//
// Override with the WASMTIME_GO_TEST_HTML_FETCHER env var. The component
// can be built via the node-jco-html sample in the sibling wasm-lab repo:
//
//	cd <wasm-lab>/node-jco-html/guest && cargo component build \
//	    --release --target wasm32-wasip2
//
// then point the env var at
// `target/wasm32-wasip2/release/html_fetcher.wasm`.
func htmlFetcherWasmPath() string {
	if p := os.Getenv("WASMTIME_GO_TEST_HTML_FETCHER"); p != "" {
		return p
	}
	return ""
}

func TestComponentNew_RejectsCoreModule(t *testing.T) {
	cfg := NewConfig()
	cfg.SetWasmComponentModel(true)
	engine := NewEngineWithConfig(cfg)

	// minimal core module preamble: not a component
	core := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	if _, err := NewComponent(engine, core); err == nil {
		t.Fatal("expected NewComponent to reject core module preamble")
	}
}

func TestComponentNew_AcceptsComponentBinary(t *testing.T) {
	path := htmlFetcherWasmPath()
	if path == "" {
		t.Skip("WASMTIME_GO_TEST_HTML_FETCHER unset; skipping")
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read wasm: %v", err)
	}

	cfg := NewConfig()
	cfg.SetWasmComponentModel(true)
	engine := NewEngineWithConfig(cfg)

	c, err := NewComponent(engine, bytes)
	if err != nil {
		t.Fatalf("NewComponent: %v", err)
	}
	c.Close()
}

func TestComponentLinker_WasiHttp_Smoke(t *testing.T) {
	if testing.Short() {
		t.Skip("network test")
	}
	path := htmlFetcherWasmPath()
	if path == "" {
		t.Skip("WASMTIME_GO_TEST_HTML_FETCHER unset; skipping")
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read wasm: %v", err)
	}

	cfg := NewConfig()
	cfg.SetWasmComponentModel(true)
	engine := NewEngineWithConfig(cfg)

	store := NewStore(engine)
	wasiCfg := NewWasiConfig()
	wasiCfg.InheritStderr()
	store.SetWasi(wasiCfg)
	store.SetWasiHttp()

	component, err := NewComponent(engine, bytes)
	if err != nil {
		t.Fatalf("NewComponent: %v", err)
	}
	defer component.Close()

	linker := NewComponentLinker(engine)
	defer linker.Close()
	if err := linker.DefineWasi(); err != nil {
		t.Fatalf("DefineWasi: %v", err)
	}
	if err := linker.DefineWasiHttp(); err != nil {
		t.Fatalf("DefineWasiHttp: %v", err)
	}

	inst, err := linker.Instantiate(store, component)
	if err != nil {
		t.Fatalf("Instantiate: %v", err)
	}

	fn := inst.GetFunc(store, "extract")
	if fn == nil {
		t.Fatal("missing export: extract")
	}

	args := []ComponentVal{ComponentValString("https://example.com")}
	results, err := fn.Call(store, args, 1)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}

	// expect: result<page-info, string>
	r := results[0]
	if r.Kind() != ComponentValKindResult {
		t.Fatalf("want result kind, got %v", r.Kind())
	}
	res := r.Result()
	if res == nil {
		t.Fatal("nil result payload")
	}
	if !res.IsOk {
		t.Fatalf("expected Ok, got Err: %+v", res.Val)
	}
	if res.Val == nil {
		t.Fatal("Ok payload missing")
	}
	if res.Val.Kind() != ComponentValKindRecord {
		t.Fatalf("want record, got %v", res.Val.Kind())
	}

	var title string
	var linkCount int
	for _, e := range res.Val.Record() {
		switch e.Name {
		case "title":
			if e.Val.Kind() != ComponentValKindString {
				t.Fatalf("title not string: %v", e.Val.Kind())
			}
			title = e.Val.String()
		case "links":
			if e.Val.Kind() != ComponentValKindList {
				t.Fatalf("links not list: %v", e.Val.Kind())
			}
			linkCount = len(e.Val.List())
		}
	}

	if title != "Example Domain" {
		t.Errorf("title = %q, want %q", title, "Example Domain")
	}
	if linkCount == 0 {
		t.Error("expected at least one link")
	}
	t.Logf("title=%q, %d link(s)", title, linkCount)
}
