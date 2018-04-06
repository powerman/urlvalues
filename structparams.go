package urlvalues

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/form"
)

// constraint describe properties of url.Values key corresponding to some value
// in target data structure.
type constraint struct {
	alias    string // shortest of all aliases
	required bool   // true for fields tagged `form:",required"`
	list     bool   // true for array or slice
	cap      []int  // cap(array) or SetMaxArraySize(10000) for slices
}

var (
	paramsCacheMu sync.Mutex
	paramsCache   = make(map[DecoderOpts]map[reflect.Type]map[string]*constraint)
)

// paramsForStruct introspect given structure and return list of all
// url.Values keys corresponding to this structure and their constraints.
func paramsForStruct(opts DecoderOpts, typ reflect.Type) (params map[string]*constraint) {
	paramsCacheMu.Lock()
	if paramsCache[opts] == nil {
		paramsCache[opts] = make(map[reflect.Type]map[string]*constraint)
	}
	if paramsCache[opts][typ] != nil {
		params = paramsCache[opts][typ]
	}
	paramsCacheMu.Unlock()
	if params != nil {
		return params
	}

	params = make(map[string]*constraint)
	addStruct(opts, typ, "", nil, nil, make(map[string]*constraint), params)

	paramsCacheMu.Lock()
	paramsCache[opts][typ] = params
	paramsCacheMu.Unlock()
	return params
}

// addStruct add given structure's fields to params.
//
// Parameters namePfx, idxPfx and byIndex are used internally for recursion only.
func addStruct(opts DecoderOpts, typ reflect.Type, namePfx string, idxPfx, cap []int, byIndex, params map[string]*constraint) {
	seen := make(map[string]bool, typ.NumField())
	typ.FieldByNameFunc(func(shortname string) bool {
		if seen[shortname] { // we'll handle recursion to anon field manually
			return false
		}
		seen[shortname] = true

		field, _ := typ.FieldByName(shortname)
		if field.PkgPath != "" { // not exported
			return false
		}

		var required bool
		tag := strings.Split(field.Tag.Get(opts.TagName), ",")
		if opts.Mode == form.ModeExplicit && len(tag) == 1 && tag[0] == "" {
			return false
		}
		if tag[0] == "-" {
			return false
		}
		if tag[0] != "" {
			shortname = tag[0]
		}
		for i := 1; i < len(tag); i++ {
			if tag[i] == "required" {
				required = true
			}
		}

		name := namePfx + shortname
		index := append(idxPfx, field.Index...)
		addElem(opts, field.Type, required, name, index, cap, byIndex, params)

		return false
	})
}

// addElem add single value of any supported type to params.
//
// Parameters name, index and byIndex are used internally for recursion only.
func addElem(opts DecoderOpts, typ reflect.Type, required bool, name string, index, cap []int, byIndex, params map[string]*constraint) { //nolint:gocyclo
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface:
		return
	case reflect.Struct: // TODO && no custom handler
		addStruct(opts, typ, name+".", index, cap, byIndex, params)
		return
	case reflect.Map:
		name += "[key]"
		if complexElem(typ) {
			index = append(index, -1)
			addElem(opts, typ.Elem(), false, name, index, cap, byIndex, params)
			return
		}
	case reflect.Array, reflect.Slice:
		if typ.Kind() == reflect.Array {
			cap = append(cap, typ.Len())
		} else if typ.Elem().Kind() != reflect.Uint8 { // exclude []byte
			cap = append(cap, opts.MaxArraySize)
		}
		if complexElem(typ) {
			name += "[idx]"
			index = append(index, -1)
			addElem(opts, typ.Elem(), false, name, index, cap, byIndex, params)
			return
		}
	}

	idx := fmt.Sprint(index)
	if byIndex[idx] == nil {
		list := typ.Kind() == reflect.Array ||
			typ.Kind() == reflect.Slice && typ.Elem().Kind() != reflect.Uint8 // exclude []byte
		byIndex[idx] = &constraint{
			alias:    name,
			required: required,
			list:     list,
			cap:      cap,
		}
	} else if len(name) < len(byIndex[idx].alias) || len(name) == len(byIndex[idx].alias) && name < byIndex[idx].alias {
		byIndex[idx].alias = name
	}
	params[name] = byIndex[idx]
	if params[name].list {
		params[name+"[idx]"] = byIndex[idx]
	}
}

func complexElem(typ reflect.Type) bool {
	typ = typ.Elem()
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface:
		return true
	case reflect.Struct, reflect.Map, reflect.Array, reflect.Slice:
		return true
	default:
		return false
	}
}
