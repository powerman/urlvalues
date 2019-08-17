// Package urlvalues implements strict decoding of url.Values to given
// struct.
//
//	WARNING: This package is experimental, API will change!
//
// It is safe for concurrent use by multiple goroutines.
//
// Strict validation rules
//
//	- (optional) error on unknown param
//	  - including param matching real, but not qualified enough field name:
//	    - struct without .field (TODO in case custom handler not registered)
//	    - map without [key]
//	- error on array overflow
//	    - array with out-of-bound [index]
//	    - too many params for array field
//	- error on scalar overflow
//	  - multiple values for non-slice/array field
//	  - multiple values for same `array[index]` or `map[key]` (in case this
//	    array/map doesn't have values of slice/array type)
//	- error on no values for non-pointer/slice/array field tagged
//	  `form:"…,required"`
//	- panic on unknown `form:""` tag option
package urlvalues

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/form"
)

// Errs contain Decode errors.
//
// Errs key can be "-" or pattern for corresponding Decode param values key.
//
// Key "-" will contain all keys from Decode param values which are not
// correspond to any of Decode param v field and thus can't be decoded.
// This key won't exists if IgnoreUnknown option is used.
//
// Pattern is same as values key with map key names replaced with [key] and
// array/slice indices replaced with [idx].
// Example: if Decode was called with this key in values
//	FieldA.MapField[something].SliceOfSliceField[42][1].FieldB
// then related key in Errs will be
//	FieldA.MapField[key].SliceOfSliceField[idx][idx].FieldB
type Errs struct{ url.Values }

func newErrs() Errs { return Errs{Values: make(url.Values)} }

// Error return all errors at once using errs.Encode.
//
// This is suitable for debugging but not for production error message.
func (errs Errs) Error() string { return errs.Encode() }

// Any returns one of available errors or empty string if there are no errors.
func (errs Errs) Any() (pattern string) {
	for pattern = range errs.Values {
		return pattern
	}
	return ""
}

// StrictDecoder wraps https://godoc.org/github.com/go-playground/form#Decoder
// to add strict validation of url.Values and normalize returned errors.
//
// Differences from wrapped form.Decoder:
//	- Decoding to field with interface type is not supported.
//	- To make field required (meaning url.Values must contain any value for
//	  this field, including empty string) tag field with:
//		`form:"…,required"`
type StrictDecoder struct {
	decoder       *form.Decoder
	decoderOpts   decoderOpts
	ignoreUnknown bool
}

// StrictDecoderOption is for internal use only and exported just to make
// golint happy.
type StrictDecoderOption func(*StrictDecoder)

// NewStrictDecoder returns new StrictDecoder.
//
// It's recommended to create one instance (for each opts) and reuse it to
// enable caching.
func NewStrictDecoder(opts ...StrictDecoderOption) *StrictDecoder {
	d := &StrictDecoder{decoder: form.NewDecoder()}
	defaults := newDecoderOpts()
	MaxArraySize(defaults.maxArraySize)(d)
	Mode(defaults.mode)(d)
	TagName(defaults.tagName)(d)
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// MaxArraySize return an option for NewStrictDecoder.
//
// See https://godoc.org/github.com/go-playground/form#Decoder.SetMaxArraySize
func MaxArraySize(size uint) StrictDecoderOption {
	return StrictDecoderOption(func(d *StrictDecoder) {
		d.decoderOpts.maxArraySize = size
		d.decoder.SetMaxArraySize(size)
	})
}

// Mode return an option for NewStrictDecoder.
//
// See https://godoc.org/github.com/go-playground/form#Decoder.SetMode
func Mode(mode form.Mode) StrictDecoderOption {
	return StrictDecoderOption(func(d *StrictDecoder) {
		d.decoderOpts.mode = mode
		d.decoder.SetMode(mode)
	})
}

// TagName return an option for NewStrictDecoder.
//
// See https://godoc.org/github.com/go-playground/form#Decoder.SetTagName
func TagName(tagName string) StrictDecoderOption {
	return StrictDecoderOption(func(d *StrictDecoder) {
		d.decoderOpts.tagName = tagName
		d.decoder.SetTagName(tagName)
	})
}

// IgnoreUnknown return an option for NewStrictDecoder.
//
// With this option Decode won't return errors related to unknown keys in
// url.Values.
func IgnoreUnknown() StrictDecoderOption {
	return StrictDecoderOption(func(d *StrictDecoder) {
		d.ignoreUnknown = true
	})
}

var typTime = reflect.TypeOf(time.Time{})

// Decode will decode values to v (which must be a pointer to a struct).
//
// It'll normalize form.Decoder panics and errors and return nil or Errs.
// It will panic if called with wrong v, but never panics on wrong values.
func (d *StrictDecoder) Decode(v interface{}, values url.Values) error { //nolint:gocyclo
	if values == nil {
		panic("data must not be nil")
	}
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct || val.Elem().Type() == typTime {
		panic("v must be a non-nil pointer to a struct")
	}

	errs := d.validate(val.Elem().Type(), values)
	if d.ignoreUnknown && errs.Values["-"] != nil {
		if len(errs.Values) == 1 {
			// Hide unknown keys from form.Decoder to avoid panic on unmatched brackets.
			orig := values
			values = make(url.Values)
			for key, value := range orig {
				values[key] = value
			}
			for _, key := range errs.Values["-"] {
				delete(values, key)
			}
		}
		delete(errs.Values, "-")
	}
	if len(errs.Values) > 0 {
		return errs
	}

	err := d.decode(v, values)
	switch err := err.(type) {
	case nil:
		return nil
	case form.DecodeErrors:
		for field, err := range err {
			msg := err.Error()
			switch {
			case strings.Contains(msg, "Map Key"):
				panic(err) // wrong v
			case strings.Contains(msg, "SetMaxArraySize"):
				panic(err) // wrong v or MaxArraySize
			case strings.Contains(msg, "Array index"):
				panic(err) // never here (should be handled by validate)
			default:
				errs.Add(field, "wrong type")
			}
		}
		return errs
	case *form.InvalidDecoderError:
		panic(err) // never here (wrong v, should be handled by panics above)
	default:
		panic(err) // never here (unmatched brackets in param name, should be handled by validate)
	}
}

func (d *StrictDecoder) decode(v interface{}, values url.Values) (err error) {
	defer func() {
		if msg := recover(); msg != nil {
			err = fmt.Errorf("%v", msg) // unmatched brackets in param name
		}
	}()
	return d.decoder.Decode(v, values)
}

func (d *StrictDecoder) validate(typ reflect.Type, values url.Values) Errs { //nolint:gocyclo
	errs := newErrs()
	params := paramsForStruct(d.decoderOpts, typ)

	// Copy values to be able to delete already processed.
	valuesCount := make(map[string]int, len(values))
	for key, val := range values {
		valuesCount[key] = len(val)
	}

	type lvalueState struct {
		firstAlias string // used to disallow multiple aliases in values
		required   bool   // used to detect missing values
	}
	lvalue := make(map[string]*lvalueState, len(params))

	for pattern, c := range params {
		if lvalue[c.alias] == nil {
			lvalue[c.alias] = &lvalueState{
				required: c.required,
			}
		}

		found := false
		if strings.ContainsRune(pattern, '[') {
			list := c.list && !(strings.HasSuffix(pattern, "[idx]") &&
				params[pattern] == params[strings.TrimSuffix(pattern, "[idx]")])
			re := compilePattern(pattern)
			for name, count := range valuesCount {
				idx := re.FindStringSubmatch(name)
				if len(idx) == 0 {
					continue
				}

				found = true
				delete(valuesCount, name)

				for i := 0; i < len(idx)-1; i++ {
					index, err := strconv.Atoi(idx[i+1])
					if err != nil {
						panic(err)
					}
					if index >= c.maxsize[i] {
						errs.Add(pattern, "index out-of-bounds")
					}
				}
				if count > 1 {
					if !list {
						errs.Add(pattern, "multiple values")
					} else if count > c.maxsize[len(c.maxsize)-1] {
						errs.Add(pattern, "too many values")
					}
				}
			}
		} else if count, ok := valuesCount[pattern]; ok {
			found = true
			delete(valuesCount, pattern)

			if count > 1 {
				if !c.list {
					errs.Add(pattern, "multiple values")
				} else if count > c.maxsize[len(c.maxsize)-1] {
					errs.Add(pattern, "too many values")
				}
			}
		}

		if found {
			if lvalue[c.alias].firstAlias == "" {
				lvalue[c.alias].firstAlias = pattern
			} else if lvalue[c.alias].firstAlias != pattern {
				first := lvalue[c.alias].firstAlias
				if first+"[idx]" != pattern && first != pattern+"[idx]" {
					errs.Add(c.alias, "multiple names for same value")
				}
			}
		}
	}

	for pattern, state := range lvalue {
		if state.required && state.firstAlias == "" {
			errs.Add(pattern, "required")
		}
	}

	for name := range valuesCount {
		errs.Add("-", name)
	}

	return errs
}

var (
	rePatternToken = regexp.MustCompile(`[^\[]+|\[idx\]|\[key\]`)
	rePattern      = regexp.MustCompile(`\A(` + rePatternToken.String() + `)+\z`)
	patternCacheMu sync.RWMutex
	patternCache   = make(map[string]*regexp.Regexp)
)

func compilePattern(pattern string) *regexp.Regexp {
	patternCacheMu.RLock()
	re := patternCache[pattern]
	patternCacheMu.RUnlock()
	if re != nil {
		return re
	}

	if !rePattern.MatchString(pattern) {
		panic("bad pattern: " + pattern)
	}
	var b strings.Builder
	_, _ = b.WriteString(`\A`)
	for _, token := range rePatternToken.FindAllStringSubmatch(pattern, -1) {
		switch token[0] {
		case "[key]":
			_, _ = b.WriteString(`\[[^\]]+\]`)
		case "[idx]":
			_, _ = b.WriteString(`\[(\d+)\]`)
		default:
			_, _ = b.WriteString(regexp.QuoteMeta(token[0]))
		}
	}
	_, _ = b.WriteString(`\z`)
	re = regexp.MustCompile(b.String())

	patternCacheMu.Lock()
	patternCache[pattern] = re
	patternCacheMu.Unlock()
	return re
}
