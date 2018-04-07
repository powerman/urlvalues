// Package urlvalues implements strict decoding of url.Values to given
// struct.
//
//  WARNING: This package is experimental, API will change!
//
// To make field required (meaning url.Values must contain any value for this
// field, including empty string) tag field with:
//   `form:"…,required"`
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

var typTime = reflect.TypeOf(time.Time{})

// Errs contain all errors from either strict validation of url.Values or
// decoding url.Values to struct.
//
// Key is pattern for related url.Values key (pattern is same as key with
// map keys replaced with [key] and array/slice indices replaced with
// [idx]).
type Errs struct{ url.Values }

func newErrs() Errs { return Errs{Values: make(url.Values)} }

// Error will return url.Values.Encode suitable for debugging but not for
// production error message.
func (errs Errs) Error() string { return errs.Encode() }

// DecoderOpts contain options for github.com/go-playground/form.Decoder.
type DecoderOpts struct {
	MaxArraySize int
	Mode         form.Mode
	TagName      string
}

// NewDecoderOpts return DecoderOpts with default values.
func NewDecoderOpts() DecoderOpts {
	return DecoderOpts{
		MaxArraySize: 10000,
		Mode:         form.ModeImplicit,
		TagName:      "form",
	}
}

// StrictDecoder adds strict validation of url.Values before decoding
// url.Values to struct.
type StrictDecoder struct {
	ignoreUnknown bool
	decoder       *form.Decoder
	decoderOpts   DecoderOpts
}

// NewStrictDecoder returns new StrictDecoder.
//
// If ignoreUnknown is true then Unmarshal will ignore unknown keys in
// url.Values.
//
// It's recommended to create one instance (for each decoderOpts) and keep
// it to enable caching.
func NewStrictDecoder(ignoreUnknown bool, decoderOpts DecoderOpts) *StrictDecoder {
	d := &StrictDecoder{
		ignoreUnknown: ignoreUnknown,
		decoder:       form.NewDecoder(),
		decoderOpts:   decoderOpts,
	}
	d.decoder.SetMaxArraySize(uint(decoderOpts.MaxArraySize))
	d.decoder.SetMode(decoderOpts.Mode)
	d.decoder.SetTagName(decoderOpts.TagName)
	return d
}

// Unmarshal will do strict validation of url.Values and decode url.Values
// to struct. It'll also normalize panics and
// decoder errors and return either usual error (in case error
// is not related to any of existing struct field) or Errs.
//
// NOTE: Do not support field/array/slice/map of interface type.
func (d *StrictDecoder) Unmarshal(values url.Values, v interface{}) (err error) { //nolint:gocyclo
	if values == nil {
		panic("data must not be nil")
	}
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct || val.Elem().Type() == typTime {
		panic("v must be a non-nil pointer to a struct")
	}

	err = d.strict(val.Elem().Type(), values)
	if err != nil {
		return err
	}

	err = d.decode(v, values)
	if err != nil {
		switch err := err.(type) {
		case form.DecodeErrors:
			errs := newErrs()
			for field, err := range err {
				msg := err.Error()
				if strings.Contains(msg, "Map Key") {
					panic(err)
				} else if strings.Contains(msg, "SetMaxArraySize") {
					panic(err)
				} else if strings.Contains(msg, "Array index") {
					errs.Add(field, "wrong index")
				} else {
					errs.Add(field, "wrong type")
				}
			}
			return errs
		case *form.InvalidDecoderError:
			panic(err) // never here (should be handled by panics above)
		default:
			return err // unmatched brackets in param name
		}
	}

	return nil
}

func (d *StrictDecoder) decode(v interface{}, values url.Values) (err error) {
	defer func() {
		if msg := recover(); msg != nil {
			err = fmt.Errorf("%v", msg)
		}
	}()
	return d.decoder.Decode(v, values)
}

func (d *StrictDecoder) strict(typ reflect.Type, values url.Values) error { //nolint:gocyclo
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

	errs := newErrs()

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
					if index >= c.cap[i] {
						errs.Add(pattern, "index out-of-bounds")
					}
				}
				if count > 1 {
					if !list {
						errs.Add(pattern, "multiple values")
					} else if count > c.cap[len(c.cap)-1] {
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
				} else if count > c.cap[len(c.cap)-1] {
					errs.Add(pattern, "too many values")
				}
			}
		}

		if found {
			if lvalue[c.alias].firstAlias == "" {
				lvalue[c.alias].firstAlias = pattern
			} else if lvalue[c.alias].firstAlias != pattern {
				errs.Add(c.alias, "multiple names for same value")
			}
		}
	}

	for pattern, state := range lvalue {
		if state.required && state.firstAlias == "" {
			errs.Add(pattern, "required")
		}
	}

	if len(errs.Values) > 0 {
		return errs
	}

	if !d.ignoreUnknown {
		for name := range valuesCount {
			return fmt.Errorf("unknown name: %q", name)
		}
	}

	return nil
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
