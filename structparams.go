package urlvalues

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/form"
)

// decoderOpts contain options for form.Decoder.
type decoderOpts struct {
	maxArraySize uint
	mode         form.Mode
	tagName      string
}

// newDecoderOpts return decoderOpts with default values.
func newDecoderOpts() decoderOpts {
	return decoderOpts{
		maxArraySize: 10000,
		mode:         form.ModeImplicit,
		tagName:      "form",
	}
}

// constraint describe properties of url.Values key corresponding to some value
// in target data structure.
type constraint struct {
	alias    string // shortest of all aliases
	required bool   // true for fields tagged `form:",required"`
	list     bool   // true for array or slice
	maxsize  []int  // maxsize(array) or SetMaxArraySize(10000) for slices
}

var (
	paramsCacheMu sync.Mutex
	paramsCache   = make(map[decoderOpts]map[reflect.Type]map[string]*constraint)
)

// paramsForStruct introspect given structure and return list of all
// url.Values keys corresponding to this structure and their constraints.
func paramsForStruct(opts decoderOpts, typ reflect.Type) (params map[string]*constraint) {
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
func addStruct(opts decoderOpts, typ reflect.Type, namePfx string, idxPfx, maxsize []int, byIndex, params map[string]*constraint) { // nolint:gocyclo
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
		tag := strings.Split(field.Tag.Get(opts.tagName), ",")
		if opts.mode == form.ModeExplicit && len(tag) == 1 && tag[0] == "" {
			return false
		}
		if tag[0] == "-" {
			return false
		}
		if tag[0] != "" {
			shortname = tag[0]
		}
		for _, opt := range tag[1:] {
			switch opt {
			case "required":
				required = true
			case "", "omitempty":
			default:
				panic(fmt.Sprintf("unknown tag option %q on field %q", opt, field.Name))
			}
		}

		name := namePfx + shortname
		index := append(idxPfx, field.Index...)
		addElem(opts, field.Type, required, name, index, maxsize, byIndex, params)

		return false
	})
}

// addElem add single value of any supported type to params.
//
// Parameters name, index and byIndex are used internally for recursion only.
func addElem(opts decoderOpts, typ reflect.Type, required bool, name string, index, maxsize []int, byIndex, params map[string]*constraint) { //nolint:gocyclo
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface:
		return
	case reflect.Struct: // TODO && no custom handler
		addStruct(opts, typ, name+".", index, maxsize, byIndex, params)
		return
	case reflect.Map:
		name += "[key]"
		if complexElem(typ) {
			index = append(index, -1)
			addElem(opts, typ.Elem(), false, name, index, maxsize, byIndex, params)
			return
		}
	case reflect.Array, reflect.Slice:
		if typ.Kind() == reflect.Array {
			maxsize = append(maxsize, typ.Len())
		} else {
			maxsize = append(maxsize, int(opts.maxArraySize))
		}
		if complexElem(typ) {
			name += "[idx]"
			index = append(index, -1)
			addElem(opts, typ.Elem(), false, name, index, maxsize, byIndex, params)
			return
		}
	}

	idx := fmt.Sprint(index)
	if byIndex[idx] == nil {
		list := typ.Kind() == reflect.Array || typ.Kind() == reflect.Slice
		byIndex[idx] = &constraint{
			alias:    name,
			required: required,
			list:     list,
			maxsize:  maxsize,
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
