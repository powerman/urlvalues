# urlvalues [![GoDoc](https://godoc.org/github.com/powerman/urlvalues?status.svg)](http://godoc.org/github.com/powerman/urlvalues) [![Go Report Card](https://goreportcard.com/badge/github.com/powerman/urlvalues)](https://goreportcard.com/report/github.com/powerman/urlvalues) [![CircleCI](https://circleci.com/gh/powerman/urlvalues.svg?style=svg)](https://circleci.com/gh/powerman/urlvalues) [![Coverage Status](https://coveralls.io/repos/github/powerman/urlvalues/badge.svg?branch=master)](https://coveralls.io/github/powerman/urlvalues?branch=master)

Go package for strict decoding of url.Values to given struct.

**WARNING:** This package is experimental, API will change!

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
- panic on unknown `form:""` tag option

## Benchmark

- `Small`/`Large` means size of struct.
- `Failure` means failed strict validation and skipped
  `form.Decoder.Decode()`.
- `Loose` means without strict validation, i.e. just
  `form.Decoder.Decode()`.

```
BenchmarkSmallFailure      	 1000000	      1212 ns/op	     944 B/op	      10 allocs/op
BenchmarkSmallSuccess      	 1000000	      1535 ns/op	     592 B/op	       9 allocs/op
BenchmarkSmallSuccessLoose 	 3000000	       519 ns/op	     448 B/op	       5 allocs/op
BenchmarkLargeFailure      	   50000	     33816 ns/op	    5556 B/op	      45 allocs/op
BenchmarkLargeSuccess      	   10000	    424874 ns/op	  817796 B/op	      77 allocs/op
BenchmarkLargeSuccessLoose 	   10000	    378008 ns/op	  814315 B/op	      50 allocs/op
```
