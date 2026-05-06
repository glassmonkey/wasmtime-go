package wasmtime

// #include <stdlib.h>
// #include <string.h>
// #include <wasm.h>
// #include <wasmtime/component/val.h>
import "C"
import (
	"fmt"
	"unsafe"
)

// ComponentValKind is the dynamic type tag for [ComponentVal]. It mirrors the
// `wasmtime_component_valkind_t` enum from the C API.
type ComponentValKind int

const (
	ComponentValKindBool ComponentValKind = iota
	ComponentValKindS8
	ComponentValKindU8
	ComponentValKindS16
	ComponentValKindU16
	ComponentValKindS32
	ComponentValKindU32
	ComponentValKindS64
	ComponentValKindU64
	ComponentValKindF32
	ComponentValKindF64
	ComponentValKindChar
	ComponentValKindString
	ComponentValKindList
	ComponentValKindRecord
	ComponentValKindTuple
	ComponentValKindVariant
	ComponentValKindEnum
	ComponentValKindOption
	ComponentValKindResult
	ComponentValKindFlags
	ComponentValKindResource
	ComponentValKindMap
)

// ComponentVal is a runtime value passed to or from a [ComponentFunc].
//
// Phase 1 of the Go bindings supports the following kinds:
//
//   - Primitives: Bool, S8/U8, S16/U16, S32/U32, S64/U64, F32, F64, Char
//   - String
//   - List, Record, Tuple
//   - Option, Result
//
// The remaining kinds (Variant, Enum, Flags, Resource, Map) currently return
// an error during conversion. Resource support requires further design around
// `wasmtime_component_resource_*` lifetimes.
type ComponentVal struct {
	kind  ComponentValKind
	bool_ bool
	i8    int8
	u8    uint8
	i16   int16
	u16   uint16
	i32   int32
	u32   uint32
	i64   int64
	u64   uint64
	f32   float32
	f64   float64
	char  rune
	str   string
	list  []ComponentVal
	rec   []ComponentValRecordEntry
	opt   *ComponentVal // nil means none
	res   *ComponentValResult
}

// ComponentValRecordEntry is a single (name, value) pair within a record.
type ComponentValRecordEntry struct {
	Name string
	Val  ComponentVal
}

// ComponentValResult is the payload of a `result<ok, err>` value.
type ComponentValResult struct {
	IsOk bool
	// Val carries the inner value. Either Ok or Err depending on IsOk. May
	// be nil if the corresponding side is unit (`result` with no payload).
	Val *ComponentVal
}

// Kind returns the dynamic type tag.
func (v ComponentVal) Kind() ComponentValKind { return v.kind }

// Constructors -----------------------------------------------------------

func ComponentValBool(b bool) ComponentVal {
	return ComponentVal{kind: ComponentValKindBool, bool_: b}
}
func ComponentValS32(n int32) ComponentVal {
	return ComponentVal{kind: ComponentValKindS32, i32: n}
}
func ComponentValU32(n uint32) ComponentVal {
	return ComponentVal{kind: ComponentValKindU32, u32: n}
}
func ComponentValS64(n int64) ComponentVal {
	return ComponentVal{kind: ComponentValKindS64, i64: n}
}
func ComponentValU64(n uint64) ComponentVal {
	return ComponentVal{kind: ComponentValKindU64, u64: n}
}
func ComponentValF32(n float32) ComponentVal {
	return ComponentVal{kind: ComponentValKindF32, f32: n}
}
func ComponentValF64(n float64) ComponentVal {
	return ComponentVal{kind: ComponentValKindF64, f64: n}
}
func ComponentValString(s string) ComponentVal {
	return ComponentVal{kind: ComponentValKindString, str: s}
}
func ComponentValList(items []ComponentVal) ComponentVal {
	return ComponentVal{kind: ComponentValKindList, list: items}
}
func ComponentValRecord(entries []ComponentValRecordEntry) ComponentVal {
	return ComponentVal{kind: ComponentValKindRecord, rec: entries}
}
func ComponentValTuple(items []ComponentVal) ComponentVal {
	return ComponentVal{kind: ComponentValKindTuple, list: items}
}
func ComponentValSome(inner ComponentVal) ComponentVal {
	return ComponentVal{kind: ComponentValKindOption, opt: &inner}
}
func ComponentValNone() ComponentVal {
	return ComponentVal{kind: ComponentValKindOption, opt: nil}
}
func ComponentValOk(inner *ComponentVal) ComponentVal {
	return ComponentVal{kind: ComponentValKindResult, res: &ComponentValResult{IsOk: true, Val: inner}}
}
func ComponentValErr(inner *ComponentVal) ComponentVal {
	return ComponentVal{kind: ComponentValKindResult, res: &ComponentValResult{IsOk: false, Val: inner}}
}

// Accessors --------------------------------------------------------------

func (v ComponentVal) Bool() bool                              { return v.bool_ }
func (v ComponentVal) S32() int32                              { return v.i32 }
func (v ComponentVal) U32() uint32                             { return v.u32 }
func (v ComponentVal) S64() int64                              { return v.i64 }
func (v ComponentVal) U64() uint64                             { return v.u64 }
func (v ComponentVal) F32() float32                            { return v.f32 }
func (v ComponentVal) F64() float64                            { return v.f64 }
func (v ComponentVal) String() string                          { return v.str }
func (v ComponentVal) List() []ComponentVal                    { return v.list }
func (v ComponentVal) Record() []ComponentValRecordEntry       { return v.rec }
func (v ComponentVal) Tuple() []ComponentVal                   { return v.list }
func (v ComponentVal) Option() (*ComponentVal, bool)           { return v.opt, v.opt != nil }
func (v ComponentVal) Result() *ComponentValResult             { return v.res }

// Conversion to / from C -------------------------------------------------

// componentValKindCEncode converts a Go kind to the C kind constant.
func componentValKindCEncode(k ComponentValKind) C.wasmtime_component_valkind_t {
	return C.wasmtime_component_valkind_t(k)
}

// componentValKindCDecode converts a C kind to Go.
func componentValKindCDecode(k C.wasmtime_component_valkind_t) ComponentValKind {
	return ComponentValKind(k)
}

// toCVal serializes a Go ComponentVal into the provided C struct in place.
//
// The caller is responsible for eventually calling
// `wasmtime_component_val_delete` on the populated C struct to free any
// nested allocations performed here.
func (v ComponentVal) toCVal(out *C.wasmtime_component_val_t) error {
	out.kind = componentValKindCEncode(v.kind)
	switch v.kind {
	case ComponentValKindBool:
		setBoolUnion(out, v.bool_)
	case ComponentValKindS8:
		setS8Union(out, v.i8)
	case ComponentValKindU8:
		setU8Union(out, v.u8)
	case ComponentValKindS16:
		setS16Union(out, v.i16)
	case ComponentValKindU16:
		setU16Union(out, v.u16)
	case ComponentValKindS32:
		setS32Union(out, v.i32)
	case ComponentValKindU32:
		setU32Union(out, v.u32)
	case ComponentValKindS64:
		setS64Union(out, v.i64)
	case ComponentValKindU64:
		setU64Union(out, v.u64)
	case ComponentValKindF32:
		setF32Union(out, v.f32)
	case ComponentValKindF64:
		setF64Union(out, v.f64)
	case ComponentValKindChar:
		setCharUnion(out, uint32(v.char))
	case ComponentValKindString:
		setStringUnion(out, v.str)
	case ComponentValKindList, ComponentValKindTuple:
		if err := setListUnion(out, v.list, v.kind == ComponentValKindTuple); err != nil {
			return err
		}
	case ComponentValKindRecord:
		if err := setRecordUnion(out, v.rec); err != nil {
			return err
		}
	case ComponentValKindOption:
		if err := setOptionUnion(out, v.opt); err != nil {
			return err
		}
	case ComponentValKindResult:
		if err := setResultUnion(out, v.res); err != nil {
			return err
		}
	default:
		return fmt.Errorf("ComponentVal kind %d not yet supported in Go bindings", v.kind)
	}
	return nil
}

// fromCVal reads a C struct into a Go ComponentVal. The C struct is left
// untouched and remains the property of the caller.
func fromCVal(c *C.wasmtime_component_val_t) (ComponentVal, error) {
	kind := componentValKindCDecode(c.kind)
	switch kind {
	case ComponentValKindBool:
		return ComponentVal{kind: kind, bool_: getBoolUnion(c)}, nil
	case ComponentValKindS8:
		return ComponentVal{kind: kind, i8: getS8Union(c)}, nil
	case ComponentValKindU8:
		return ComponentVal{kind: kind, u8: getU8Union(c)}, nil
	case ComponentValKindS16:
		return ComponentVal{kind: kind, i16: getS16Union(c)}, nil
	case ComponentValKindU16:
		return ComponentVal{kind: kind, u16: getU16Union(c)}, nil
	case ComponentValKindS32:
		return ComponentVal{kind: kind, i32: getS32Union(c)}, nil
	case ComponentValKindU32:
		return ComponentVal{kind: kind, u32: getU32Union(c)}, nil
	case ComponentValKindS64:
		return ComponentVal{kind: kind, i64: getS64Union(c)}, nil
	case ComponentValKindU64:
		return ComponentVal{kind: kind, u64: getU64Union(c)}, nil
	case ComponentValKindF32:
		return ComponentVal{kind: kind, f32: getF32Union(c)}, nil
	case ComponentValKindF64:
		return ComponentVal{kind: kind, f64: getF64Union(c)}, nil
	case ComponentValKindChar:
		return ComponentVal{kind: kind, char: rune(getCharUnion(c))}, nil
	case ComponentValKindString:
		return ComponentVal{kind: kind, str: getStringUnion(c)}, nil
	case ComponentValKindList, ComponentValKindTuple:
		items, err := getListUnion(c)
		if err != nil {
			return ComponentVal{}, err
		}
		return ComponentVal{kind: kind, list: items}, nil
	case ComponentValKindRecord:
		entries, err := getRecordUnion(c)
		if err != nil {
			return ComponentVal{}, err
		}
		return ComponentVal{kind: kind, rec: entries}, nil
	case ComponentValKindOption:
		opt, err := getOptionUnion(c)
		if err != nil {
			return ComponentVal{}, err
		}
		return ComponentVal{kind: kind, opt: opt}, nil
	case ComponentValKindResult:
		res, err := getResultUnion(c)
		if err != nil {
			return ComponentVal{}, err
		}
		return ComponentVal{kind: kind, res: res}, nil
	default:
		return ComponentVal{}, fmt.Errorf("ComponentVal kind %d not yet supported in Go bindings", kind)
	}
}

// --- low-level union setters/getters -----------------------------------
//
// `wasmtime_component_val_t.of` is a C union. cgo doesn't expose union
// members directly so we use unsafe.Pointer reinterpretation.
// All union variants share the same starting address; we just cast to the
// appropriate type per variant.

func unionPtr(c *C.wasmtime_component_val_t) unsafe.Pointer {
	return unsafe.Pointer(&c.of)
}

func setBoolUnion(c *C.wasmtime_component_val_t, v bool) {
	*(*C.bool)(unionPtr(c)) = C.bool(v)
}
func getBoolUnion(c *C.wasmtime_component_val_t) bool {
	return bool(*(*C.bool)(unionPtr(c)))
}

func setS8Union(c *C.wasmtime_component_val_t, v int8) {
	*(*C.int8_t)(unionPtr(c)) = C.int8_t(v)
}
func getS8Union(c *C.wasmtime_component_val_t) int8 {
	return int8(*(*C.int8_t)(unionPtr(c)))
}

func setU8Union(c *C.wasmtime_component_val_t, v uint8) {
	*(*C.uint8_t)(unionPtr(c)) = C.uint8_t(v)
}
func getU8Union(c *C.wasmtime_component_val_t) uint8 {
	return uint8(*(*C.uint8_t)(unionPtr(c)))
}

func setS16Union(c *C.wasmtime_component_val_t, v int16) {
	*(*C.int16_t)(unionPtr(c)) = C.int16_t(v)
}
func getS16Union(c *C.wasmtime_component_val_t) int16 {
	return int16(*(*C.int16_t)(unionPtr(c)))
}

func setU16Union(c *C.wasmtime_component_val_t, v uint16) {
	*(*C.uint16_t)(unionPtr(c)) = C.uint16_t(v)
}
func getU16Union(c *C.wasmtime_component_val_t) uint16 {
	return uint16(*(*C.uint16_t)(unionPtr(c)))
}

func setS32Union(c *C.wasmtime_component_val_t, v int32) {
	*(*C.int32_t)(unionPtr(c)) = C.int32_t(v)
}
func getS32Union(c *C.wasmtime_component_val_t) int32 {
	return int32(*(*C.int32_t)(unionPtr(c)))
}

func setU32Union(c *C.wasmtime_component_val_t, v uint32) {
	*(*C.uint32_t)(unionPtr(c)) = C.uint32_t(v)
}
func getU32Union(c *C.wasmtime_component_val_t) uint32 {
	return uint32(*(*C.uint32_t)(unionPtr(c)))
}

func setS64Union(c *C.wasmtime_component_val_t, v int64) {
	*(*C.int64_t)(unionPtr(c)) = C.int64_t(v)
}
func getS64Union(c *C.wasmtime_component_val_t) int64 {
	return int64(*(*C.int64_t)(unionPtr(c)))
}

func setU64Union(c *C.wasmtime_component_val_t, v uint64) {
	*(*C.uint64_t)(unionPtr(c)) = C.uint64_t(v)
}
func getU64Union(c *C.wasmtime_component_val_t) uint64 {
	return uint64(*(*C.uint64_t)(unionPtr(c)))
}

func setF32Union(c *C.wasmtime_component_val_t, v float32) {
	*(*C.float)(unionPtr(c)) = C.float(v)
}
func getF32Union(c *C.wasmtime_component_val_t) float32 {
	return float32(*(*C.float)(unionPtr(c)))
}

func setF64Union(c *C.wasmtime_component_val_t, v float64) {
	*(*C.double)(unionPtr(c)) = C.double(v)
}
func getF64Union(c *C.wasmtime_component_val_t) float64 {
	return float64(*(*C.double)(unionPtr(c)))
}

func setCharUnion(c *C.wasmtime_component_val_t, v uint32) {
	*(*C.uint32_t)(unionPtr(c)) = C.uint32_t(v)
}
func getCharUnion(c *C.wasmtime_component_val_t) uint32 {
	return uint32(*(*C.uint32_t)(unionPtr(c)))
}

// --- string -------------------------------------------------------------

func setStringUnion(c *C.wasmtime_component_val_t, s string) {
	vec := (*C.wasm_name_t)(unionPtr(c))
	vec.size = C.size_t(len(s))
	if len(s) == 0 {
		vec.data = nil
		return
	}
	buf := C.malloc(C.size_t(len(s)))
	if buf == nil {
		panic("malloc failed")
	}
	src := []byte(s)
	C.memcpy(buf, unsafe.Pointer(&src[0]), C.size_t(len(src)))
	vec.data = (*C.wasm_byte_t)(buf)
}

func getStringUnion(c *C.wasmtime_component_val_t) string {
	vec := (*C.wasm_name_t)(unionPtr(c))
	if vec.size == 0 {
		return ""
	}
	return C.GoStringN((*C.char)(unsafe.Pointer(vec.data)), C.int(vec.size))
}

// --- list / tuple -------------------------------------------------------
//
// list and tuple have the same C layout (vallist / valtuple are both vecs of
// component_val_t). We pick the right field based on the kind.

func setListUnion(c *C.wasmtime_component_val_t, items []ComponentVal, isTuple bool) error {
	var vec *C.wasmtime_component_vallist_t
	if isTuple {
		vec = (*C.wasmtime_component_vallist_t)(unionPtr(c)) // same layout as valtuple
	} else {
		vec = (*C.wasmtime_component_vallist_t)(unionPtr(c))
	}
	C.wasmtime_component_vallist_new_uninit(vec, C.size_t(len(items)))
	for i, item := range items {
		entry := (*C.wasmtime_component_val_t)(unsafe.Pointer(uintptr(unsafe.Pointer(vec.data)) + uintptr(i)*unsafe.Sizeof(C.wasmtime_component_val_t{})))
		if err := item.toCVal(entry); err != nil {
			return err
		}
	}
	return nil
}

func getListUnion(c *C.wasmtime_component_val_t) ([]ComponentVal, error) {
	vec := (*C.wasmtime_component_vallist_t)(unionPtr(c))
	if vec.size == 0 {
		return nil, nil
	}
	out := make([]ComponentVal, int(vec.size))
	for i := 0; i < int(vec.size); i++ {
		entry := (*C.wasmtime_component_val_t)(unsafe.Pointer(uintptr(unsafe.Pointer(vec.data)) + uintptr(i)*unsafe.Sizeof(C.wasmtime_component_val_t{})))
		v, err := fromCVal(entry)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// --- record -------------------------------------------------------------

func setRecordUnion(c *C.wasmtime_component_val_t, entries []ComponentValRecordEntry) error {
	vec := (*C.wasmtime_component_valrecord_t)(unionPtr(c))
	C.wasmtime_component_valrecord_new_uninit(vec, C.size_t(len(entries)))
	entrySize := unsafe.Sizeof(C.wasmtime_component_valrecord_entry_t{})
	for i, e := range entries {
		raw := (*C.wasmtime_component_valrecord_entry_t)(unsafe.Pointer(uintptr(unsafe.Pointer(vec.data)) + uintptr(i)*entrySize))
		// name
		raw.name.size = C.size_t(len(e.Name))
		if len(e.Name) > 0 {
			buf := C.malloc(C.size_t(len(e.Name)))
			if buf == nil {
				panic("malloc failed")
			}
			src := []byte(e.Name)
			C.memcpy(buf, unsafe.Pointer(&src[0]), C.size_t(len(src)))
			raw.name.data = (*C.wasm_byte_t)(buf)
		} else {
			raw.name.data = nil
		}
		if err := e.Val.toCVal(&raw.val); err != nil {
			return err
		}
	}
	return nil
}

func getRecordUnion(c *C.wasmtime_component_val_t) ([]ComponentValRecordEntry, error) {
	vec := (*C.wasmtime_component_valrecord_t)(unionPtr(c))
	if vec.size == 0 {
		return nil, nil
	}
	out := make([]ComponentValRecordEntry, int(vec.size))
	entrySize := unsafe.Sizeof(C.wasmtime_component_valrecord_entry_t{})
	for i := 0; i < int(vec.size); i++ {
		raw := (*C.wasmtime_component_valrecord_entry_t)(unsafe.Pointer(uintptr(unsafe.Pointer(vec.data)) + uintptr(i)*entrySize))
		name := ""
		if raw.name.size > 0 {
			name = C.GoStringN((*C.char)(unsafe.Pointer(raw.name.data)), C.int(raw.name.size))
		}
		v, err := fromCVal(&raw.val)
		if err != nil {
			return nil, err
		}
		out[i] = ComponentValRecordEntry{Name: name, Val: v}
	}
	return out, nil
}

// --- option -------------------------------------------------------------

func setOptionUnion(c *C.wasmtime_component_val_t, inner *ComponentVal) error {
	if inner == nil {
		// represent None: set option to NULL
		*(**C.wasmtime_component_val_t)(unionPtr(c)) = nil
		return nil
	}
	// allocate via wasmtime_component_val_new
	tmp := C.wasmtime_component_val_t{}
	if err := inner.toCVal(&tmp); err != nil {
		return err
	}
	heap := C.wasmtime_component_val_new(&tmp)
	*(**C.wasmtime_component_val_t)(unionPtr(c)) = heap
	return nil
}

func getOptionUnion(c *C.wasmtime_component_val_t) (*ComponentVal, error) {
	heap := *(**C.wasmtime_component_val_t)(unionPtr(c))
	if heap == nil {
		return nil, nil
	}
	v, err := fromCVal(heap)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// --- result -------------------------------------------------------------

func setResultUnion(c *C.wasmtime_component_val_t, r *ComponentValResult) error {
	if r == nil {
		return fmt.Errorf("nil ComponentValResult")
	}
	type cResult struct {
		isOk C.bool
		val  *C.wasmtime_component_val_t
	}
	out := (*cResult)(unionPtr(c))
	out.isOk = C.bool(r.IsOk)
	if r.Val == nil {
		out.val = nil
		return nil
	}
	tmp := C.wasmtime_component_val_t{}
	if err := r.Val.toCVal(&tmp); err != nil {
		return err
	}
	out.val = C.wasmtime_component_val_new(&tmp)
	return nil
}

func getResultUnion(c *C.wasmtime_component_val_t) (*ComponentValResult, error) {
	type cResult struct {
		isOk C.bool
		val  *C.wasmtime_component_val_t
	}
	in := (*cResult)(unionPtr(c))
	r := &ComponentValResult{IsOk: bool(in.isOk)}
	if in.val == nil {
		return r, nil
	}
	v, err := fromCVal(in.val)
	if err != nil {
		return nil, err
	}
	r.Val = &v
	return r, nil
}
