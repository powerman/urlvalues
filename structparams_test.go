package urlvalues

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/go-playground/form"
	"github.com/powerman/check"
)

func TestParamsEmpty(tt *testing.T) {
	t := check.T(tt)
	var data struct{}
	t.DeepEqual(paramsForStruct(newDecoderOpts(), reflect.TypeOf(data)), map[string]*constraint{})
}

func TestParamsExported(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		I int
		a string
	}
	t.DeepEqual(paramsForStruct(newDecoderOpts(), reflect.TypeOf(data)), map[string]*constraint{
		"I": &constraint{alias: "I"},
	})
	t.Nil(form.NewDecoder().Decode(&data, url.Values{
		"I": {"42"},
		"a": {"abc"},
	}))
	t.Equal(data.I, 42)
	t.Equal(data.a, "")
}

func TestParamsModeExplicit(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		B bool `form:"b"`
		I int
		A string `form:"-"`
		X string
		Y string `form:""`
		Z string `form:","`
	}
	opts := newDecoderOpts()
	opts.mode = form.ModeExplicit
	t.DeepEqual(paramsForStruct(opts, reflect.TypeOf(data)), map[string]*constraint{
		"b": &constraint{alias: "b"},
		"Z": &constraint{alias: "Z"},
	})
	decoder := form.NewDecoder()
	decoder.SetMode(opts.mode)
	t.Nil(decoder.Decode(&data, url.Values{
		"b": {"true"},
		"I": {"42"},
		"A": {"abc"},
		"X": {"XXX"},
		"Y": {"Yes"},
		"Z": {"zxc"},
	}))
	t.Equal(data.B, true)
	t.Equal(data.I, 0)
	t.Equal(data.A, "")
	t.Equal(data.X, "")
	t.Equal(data.Y, "")
	t.Equal(data.Z, "zxc")
}

func TestParamsIgnored(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		I int
		A string `form:"-"`
	}
	t.DeepEqual(paramsForStruct(newDecoderOpts(), reflect.TypeOf(data)), map[string]*constraint{
		"I": &constraint{alias: "I"},
	})
}

func TestParamsAlias(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		I int
		A string `form:"a"`
	}
	t.DeepEqual(paramsForStruct(newDecoderOpts(), reflect.TypeOf(data)), map[string]*constraint{
		"I": &constraint{alias: "I"},
		"a": &constraint{alias: "a"},
	})
}

func TestParamsRequired(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		I int
		A string `form:",required"`
	}
	t.DeepEqual(paramsForStruct(newDecoderOpts(), reflect.TypeOf(data)), map[string]*constraint{
		"I": &constraint{alias: "I"},
		"A": &constraint{alias: "A", required: true},
	})
}

func TestParamsList(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		A  [3]int
		B1 [5]byte
		B2 []byte
		S  []int
	}
	t.DeepEqual(paramsForStruct(newDecoderOpts(), reflect.TypeOf(data)), map[string]*constraint{
		"A":       &constraint{alias: "A", list: true, cap: []int{3}},
		"A[idx]":  &constraint{alias: "A", list: true, cap: []int{3}},
		"B1":      &constraint{alias: "B1", list: true, cap: []int{5}},
		"B1[idx]": &constraint{alias: "B1", list: true, cap: []int{5}},
		"B2":      &constraint{alias: "B2"},
		"S":       &constraint{alias: "S", list: true, cap: []int{10000}},
		"S[idx]":  &constraint{alias: "S", list: true, cap: []int{10000}},
	})
}

func TestParamsPtr(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		A  *[3]*int
		I  *int
		M  *map[string]*string
		S  *[]*int
		SS *[]*int
		Z  **string
	}
	t.DeepEqual(paramsForStruct(newDecoderOpts(), reflect.TypeOf(data)), map[string]*constraint{
		"A":       &constraint{alias: "A", list: true, cap: []int{3}},
		"A[idx]":  &constraint{alias: "A", list: true, cap: []int{3}},
		"I":       &constraint{alias: "I"},
		"M[key]":  &constraint{alias: "M[key]"},
		"S":       &constraint{alias: "S", list: true, cap: []int{10000}},
		"S[idx]":  &constraint{alias: "S", list: true, cap: []int{10000}},
		"SS":      &constraint{alias: "SS", list: true, cap: []int{10000}},
		"SS[idx]": &constraint{alias: "SS", list: true, cap: []int{10000}},
		"Z":       &constraint{alias: "Z"},
	})
	t.Nil(form.NewDecoder().Decode(&data, url.Values{
		// "A": {"10"}, // panic
		// "A[2]":   {"30"}, // panic
		"I":      {"42"},
		"M[one]": {"One"},
		"S":      {"100", "200"},
		"SS[2]":  {"300"},
		"Z":      {"zxc"},
	}))
	var ival = []int{42, 100, 200, 300}
	var sval = []string{"One", "zxc"}
	var s = make([]*int, 2)
	var ss = make([]*int, 3)
	s[0] = &ival[1]
	s[1] = &ival[2]
	ss[2] = &ival[3]
	var zref = &sval[1]
	t.Nil(data.A)
	t.DeepEqual(data.I, &ival[0])
	t.DeepEqual(data.M, &map[string]*string{"one": &sval[0]})
	t.DeepEqual(data.S, &s)
	t.DeepEqual(data.SS, &ss)
	t.DeepEqual(data.Z, &zref)
}

func TestParamsNoChanFuncInterface(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		Chan chan int
		Func func(string)
		A    interface{}
		I    interface{}
		F    interface{}
	}
	var a string
	var i int
	var f struct{ N int }
	data.A = &a
	data.I = &i
	data.F = &f
	t.DeepEqual(paramsForStruct(newDecoderOpts(), reflect.TypeOf(data)), map[string]*constraint{})
	t.Nil(form.NewDecoder().Decode(&data, url.Values{
		"Chan": {"42"},
		"Func": {"arg"},
		"A":    {"abc"},
		"I":    {"42"},
		"F.N":  {"100"},
	}))
	var ival = 42
	var sval = "abc"
	t.DeepEqual(data.A, &sval)
	t.DeepEqual(data.I, &ival)
	t.DeepEqual(data.F, &struct{ N int }{N: 100})
	t.Nil(data.Chan)
	t.Nil(data.Func)
}

type (
	DataC struct {
		C []string
		Z string
	}
	DataB struct {
		B  string `form:",required"`
		M  map[int]int
		S1 map[string]DataC
		S2 map[string][]DataC
		S3 []DataC
		S4 [2][2]DataC
		DataC
		Z string `form:"zz"`
	}
	DataA struct {
		A string
		DataB
		DataC
		X string `form:"-"`
		Y struct {
			I int `validate:"min=10,max=99"`
			S string
		}
		Z string
	}
)

func TestParamsComplex(tt *testing.T) {
	t := check.T(tt)
	var data DataA
	t.DeepEqual(paramsForStruct(newDecoderOpts(), reflect.TypeOf(data)), map[string]*constraint{
		"A":                         &constraint{alias: "A"},
		"Y.I":                       &constraint{alias: "Y.I"},
		"Y.S":                       &constraint{alias: "Y.S"},
		"Z":                         &constraint{alias: "Z"},
		"DataB.B":                   &constraint{alias: "B", required: true},
		"DataB.M[key]":              &constraint{alias: "M[key]"},
		"DataB.S1[key].C":           &constraint{alias: "S1[key].C", list: true, cap: []int{10000}},
		"DataB.S1[key].C[idx]":      &constraint{alias: "S1[key].C", list: true, cap: []int{10000}},
		"DataB.S1[key].Z":           &constraint{alias: "S1[key].Z"},
		"DataB.S2[key][idx].C":      &constraint{alias: "S2[key][idx].C", list: true, cap: []int{10000, 10000}},
		"DataB.S2[key][idx].C[idx]": &constraint{alias: "S2[key][idx].C", list: true, cap: []int{10000, 10000}},
		"DataB.S2[key][idx].Z":      &constraint{alias: "S2[key][idx].Z", cap: []int{10000}},
		"DataB.S3[idx].C":           &constraint{alias: "S3[idx].C", list: true, cap: []int{10000, 10000}},
		"DataB.S3[idx].C[idx]":      &constraint{alias: "S3[idx].C", list: true, cap: []int{10000, 10000}},
		"DataB.S3[idx].Z":           &constraint{alias: "S3[idx].Z", cap: []int{10000}},
		"DataB.S4[idx][idx].C":      &constraint{alias: "S4[idx][idx].C", list: true, cap: []int{2, 2, 10000}},
		"DataB.S4[idx][idx].C[idx]": &constraint{alias: "S4[idx][idx].C", list: true, cap: []int{2, 2, 10000}},
		"DataB.S4[idx][idx].Z":      &constraint{alias: "S4[idx][idx].Z", cap: []int{2, 2}},
		"DataB.zz":                  &constraint{alias: "DataB.zz"},
		"DataB.DataC.C":             &constraint{alias: "DataB.C", list: true, cap: []int{10000}},
		"DataB.DataC.C[idx]":        &constraint{alias: "DataB.C", list: true, cap: []int{10000}},
		"DataB.DataC.Z":             &constraint{alias: "DataB.DataC.Z"},
		"DataB.C":                   &constraint{alias: "DataB.C", list: true, cap: []int{10000}},
		"DataB.C[idx]":              &constraint{alias: "DataB.C", list: true, cap: []int{10000}},
		"B":                         &constraint{alias: "B", required: true},
		"M[key]":                    &constraint{alias: "M[key]"},
		"S1[key].C":                 &constraint{alias: "S1[key].C", list: true, cap: []int{10000}},
		"S1[key].C[idx]":            &constraint{alias: "S1[key].C", list: true, cap: []int{10000}},
		"S1[key].Z":                 &constraint{alias: "S1[key].Z"},
		"S2[key][idx].C":            &constraint{alias: "S2[key][idx].C", list: true, cap: []int{10000, 10000}},
		"S2[key][idx].C[idx]":       &constraint{alias: "S2[key][idx].C", list: true, cap: []int{10000, 10000}},
		"S2[key][idx].Z":            &constraint{alias: "S2[key][idx].Z", cap: []int{10000}},
		"S3[idx].C":                 &constraint{alias: "S3[idx].C", list: true, cap: []int{10000, 10000}},
		"S3[idx].C[idx]":            &constraint{alias: "S3[idx].C", list: true, cap: []int{10000, 10000}},
		"S3[idx].Z":                 &constraint{alias: "S3[idx].Z", cap: []int{10000}},
		"S4[idx][idx].C":            &constraint{alias: "S4[idx][idx].C", list: true, cap: []int{2, 2, 10000}},
		"S4[idx][idx].C[idx]":       &constraint{alias: "S4[idx][idx].C", list: true, cap: []int{2, 2, 10000}},
		"S4[idx][idx].Z":            &constraint{alias: "S4[idx][idx].Z", cap: []int{2, 2}},
		"DataC.C":                   &constraint{alias: "C", list: true, cap: []int{10000}},
		"DataC.C[idx]":              &constraint{alias: "C", list: true, cap: []int{10000}},
		"DataC.Z":                   &constraint{alias: "DataC.Z"},
		"C":                         &constraint{alias: "C", list: true, cap: []int{10000}},
		"C[idx]":                    &constraint{alias: "C", list: true, cap: []int{10000}},
	})
	t.Nil(form.NewDecoder().Decode(&data, url.Values{
		"S2[zero][1].C[2]":      {"three"},
		"DataB.S2[one][2].C[3]": {"four"},
		"Z":  {"one"},
		"zz": {"two"},
	}))
	t.Equal(data.S2["zero"][1].C[2], "three")
	t.Equal(data.S2["one"][2].C[3], "four")
	t.Equal(data.Z, "one")
	t.Equal(data.DataB.Z, "two")
	t.Equal(data.DataC.Z, "one")
}

// This test is here to improve other tests if/when it'll be fixed.
func TestDecodeArrayBug29(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		A [2]string
	}
	t.Panic(func() {
		form.NewDecoder().Decode(&data, url.Values{
			"A":    {"10"},
			"A[1]": {"20"},
		})
	})
}

// This test is here to improve other tests if/when it'll be fixed.
func TestDecodeSliceBug30(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		A []string
	}
	form.NewDecoder().Decode(&data, url.Values{
		"A": {"10"},
	})
	t.DeepEqual(data.A, []string{"10"})
	data.A = nil
	form.NewDecoder().Decode(&data, url.Values{
		"A[1]": {"20"},
	})
	t.DeepEqual(data.A, []string{"", "20"})
	data.A = nil
	form.NewDecoder().Decode(&data, url.Values{
		"A":    {"10"},
		"A[1]": {"20"},
	})
	t.NotDeepEqual(data.A, []string{"10", "20"})
}

// This test is here to improve other tests if/when it'll be fixed.
func TestDecodeEmbeddedBug31(tt *testing.T) {
	t := check.T(tt)
	type Embed struct {
		A string
	}
	var data struct {
		A string
		Embed
	}
	t.Nil(form.NewDecoder().Decode(&data, url.Values{
		"A": {"one"},
	}))
	t.Equal(data.A, "one")
	t.Equal(data.Embed.A, "one")
}
