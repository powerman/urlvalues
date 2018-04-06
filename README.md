# urlvalues [![GoDoc](https://godoc.org/github.com/powerman/urlvalues?status.svg)](http://godoc.org/github.com/powerman/urlvalues) [![Go Report Card](https://goreportcard.com/badge/github.com/powerman/urlvalues)](https://goreportcard.com/report/github.com/powerman/urlvalues) [![CircleCI](https://circleci.com/gh/powerman/urlvalues.svg?style=svg)](https://circleci.com/gh/powerman/urlvalues) [![Coverage Status](https://coveralls.io/repos/github/powerman/urlvalues/badge.svg?branch=master)](https://coveralls.io/github/powerman/urlvalues?branch=master)

Go package for unmarshaling url.Values to struct with strict validation.

**WARNING:** This package is experimental, API will change!

In case of successful strict validation url.Values will be
decoded to given struct using https://github.com/go-playground/form and
validated using https://github.com/go-playground/validator.

To make field required (meaning url.Values must contain any value for this
field, including empty string) tag field with `form:"…,required"`.

## Strict validation rules

- error on unknown param
  - including param matching real, but not qualified enough field name:
    - struct without .field (TODO in case custom handler not registered)
    - map without [key]
    - array with out-of-bound [index]
    - too many params for array field
  - error instead of panic on field name with unmatched brackets
- error on decoding multiple values to scalar
  - multiple values for non-slice/array field (except []byte)
    - also pointer to slice/array
  - including multiple values for same `array[index]` or `map[key]`, in
    case this array/map doesn't have values of slice/array type
- error on decoding no values to non-pointer/slice/array field tagged
  `",required"`

## Benchmark

- `Small`/`Large` means size of struct.
- `Failure` means failed strict validation and skipped
  `form.Decoder.Decode()` and `validator.Validate.Struct()`.
- `Loose` means without strict validation, i.e. just
  `form.Decoder.Decode()` and `validator.Validate.Struct()`.

```
BenchmarkSmallFailure      	 1000000	      1262 ns/op	     944 B/op	      10 allocs/op
BenchmarkSmallSuccess      	 1000000	      1961 ns/op	     592 B/op	       9 allocs/op
BenchmarkSmallSuccessLoose 	 2000000	       809 ns/op	     448 B/op	       5 allocs/op
BenchmarkLargeFailure      	   30000	     44528 ns/op	    7851 B/op	     260 allocs/op
BenchmarkLargeSuccess      	   10000	    474760 ns/op	  820307 B/op	     308 allocs/op
BenchmarkLargeSuccessLoose 	   10000	    416117 ns/op	  814421 B/op	      51 allocs/op
```
