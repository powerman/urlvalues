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
		"I": {alias: "I"},
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
		"b": {alias: "b"},
		"Z": {alias: "Z"},
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
		"I": {alias: "I"},
	})
}

func TestParamsAlias(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		I int
		A string `form:"a"`
	}
	t.DeepEqual(paramsForStruct(newDecoderOpts(), reflect.TypeOf(data)), map[string]*constraint{
		"I": {alias: "I"},
		"a": {alias: "a"},
	})
}

func TestParamsRequired(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		I int
		A string `form:",required"`
	}
	t.DeepEqual(paramsForStruct(newDecoderOpts(), reflect.TypeOf(data)), map[string]*constraint{
		"I": {alias: "I"},
		"A": {alias: "A", required: true},
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
		"A":       {alias: "A", list: true, maxsize: []int{3}},
		"A[idx]":  {alias: "A", list: true, maxsize: []int{3}},
		"B1":      {alias: "B1", list: true, maxsize: []int{5}},
		"B1[idx]": {alias: "B1", list: true, maxsize: []int{5}},
		"B2":      {alias: "B2", list: true, maxsize: []int{10000}},
		"B2[idx]": {alias: "B2", list: true, maxsize: []int{10000}},
		"S":       {alias: "S", list: true, maxsize: []int{10000}},
		"S[idx]":  {alias: "S", list: true, maxsize: []int{10000}},
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
		"A":       {alias: "A", list: true, maxsize: []int{3}},
		"A[idx]":  {alias: "A", list: true, maxsize: []int{3}},
		"I":       {alias: "I"},
		"M[key]":  {alias: "M[key]"},
		"S":       {alias: "S", list: true, maxsize: []int{10000}},
		"S[idx]":  {alias: "S", list: true, maxsize: []int{10000}},
		"SS":      {alias: "SS", list: true, maxsize: []int{10000}},
		"SS[idx]": {alias: "SS", list: true, maxsize: []int{10000}},
		"Z":       {alias: "Z"},
	})
	t.Nil(form.NewDecoder().Decode(&data, url.Values{
		"A":      {"10"},
		"A[2]":   {"30"},
		"I":      {"42"},
		"M[one]": {"One"},
		"S":      {"100", "200"},
		"SS[2]":  {"300"},
		"Z":      {"zxc"},
	}))
	var ival = []int{42, 100, 200, 300, 10, 30}
	var sval = []string{"One", "zxc"}
	var aval [3]*int
	aval[0] = &ival[4]
	aval[2] = &ival[5]
	var s = make([]*int, 2)
	var ss = make([]*int, 3)
	s[0] = &ival[1]
	s[1] = &ival[2]
	ss[2] = &ival[3]
	var zref = &sval[1]
	t.DeepEqual(data.A, &aval)
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
		"A":                         {alias: "A"},
		"Y.I":                       {alias: "Y.I"},
		"Y.S":                       {alias: "Y.S"},
		"Z":                         {alias: "Z"},
		"DataB.B":                   {alias: "B", required: true},
		"DataB.M[key]":              {alias: "M[key]"},
		"DataB.S1[key].C":           {alias: "S1[key].C", list: true, maxsize: []int{10000}},
		"DataB.S1[key].C[idx]":      {alias: "S1[key].C", list: true, maxsize: []int{10000}},
		"DataB.S1[key].Z":           {alias: "S1[key].Z"},
		"DataB.S2[key][idx].C":      {alias: "S2[key][idx].C", list: true, maxsize: []int{10000, 10000}},
		"DataB.S2[key][idx].C[idx]": {alias: "S2[key][idx].C", list: true, maxsize: []int{10000, 10000}},
		"DataB.S2[key][idx].Z":      {alias: "S2[key][idx].Z", maxsize: []int{10000}},
		"DataB.S3[idx].C":           {alias: "S3[idx].C", list: true, maxsize: []int{10000, 10000}},
		"DataB.S3[idx].C[idx]":      {alias: "S3[idx].C", list: true, maxsize: []int{10000, 10000}},
		"DataB.S3[idx].Z":           {alias: "S3[idx].Z", maxsize: []int{10000}},
		"DataB.S4[idx][idx].C":      {alias: "S4[idx][idx].C", list: true, maxsize: []int{2, 2, 10000}},
		"DataB.S4[idx][idx].C[idx]": {alias: "S4[idx][idx].C", list: true, maxsize: []int{2, 2, 10000}},
		"DataB.S4[idx][idx].Z":      {alias: "S4[idx][idx].Z", maxsize: []int{2, 2}},
		"DataB.zz":                  {alias: "DataB.zz"},
		"DataB.DataC.C":             {alias: "DataB.C", list: true, maxsize: []int{10000}},
		"DataB.DataC.C[idx]":        {alias: "DataB.C", list: true, maxsize: []int{10000}},
		"DataB.DataC.Z":             {alias: "DataB.DataC.Z"},
		"DataB.C":                   {alias: "DataB.C", list: true, maxsize: []int{10000}},
		"DataB.C[idx]":              {alias: "DataB.C", list: true, maxsize: []int{10000}},
		"B":                         {alias: "B", required: true},
		"M[key]":                    {alias: "M[key]"},
		"S1[key].C":                 {alias: "S1[key].C", list: true, maxsize: []int{10000}},
		"S1[key].C[idx]":            {alias: "S1[key].C", list: true, maxsize: []int{10000}},
		"S1[key].Z":                 {alias: "S1[key].Z"},
		"S2[key][idx].C":            {alias: "S2[key][idx].C", list: true, maxsize: []int{10000, 10000}},
		"S2[key][idx].C[idx]":       {alias: "S2[key][idx].C", list: true, maxsize: []int{10000, 10000}},
		"S2[key][idx].Z":            {alias: "S2[key][idx].Z", maxsize: []int{10000}},
		"S3[idx].C":                 {alias: "S3[idx].C", list: true, maxsize: []int{10000, 10000}},
		"S3[idx].C[idx]":            {alias: "S3[idx].C", list: true, maxsize: []int{10000, 10000}},
		"S3[idx].Z":                 {alias: "S3[idx].Z", maxsize: []int{10000}},
		"S4[idx][idx].C":            {alias: "S4[idx][idx].C", list: true, maxsize: []int{2, 2, 10000}},
		"S4[idx][idx].C[idx]":       {alias: "S4[idx][idx].C", list: true, maxsize: []int{2, 2, 10000}},
		"S4[idx][idx].Z":            {alias: "S4[idx][idx].Z", maxsize: []int{2, 2}},
		"DataC.C":                   {alias: "C", list: true, maxsize: []int{10000}},
		"DataC.C[idx]":              {alias: "C", list: true, maxsize: []int{10000}},
		"DataC.Z":                   {alias: "DataC.Z"},
		"C":                         {alias: "C", list: true, maxsize: []int{10000}},
		"C[idx]":                    {alias: "C", list: true, maxsize: []int{10000}},
	})
	t.Nil(form.NewDecoder().Decode(&data, url.Values{
		"S2[zero][1].C[2]":      {"three"},
		"DataB.S2[one][2].C[3]": {"four"},
		"Z":                     {"one"},
		"zz":                    {"two"},
	}))
	t.Equal(data.S2["zero"][1].C[2], "three")
	t.Equal(data.S2["one"][2].C[3], "four")
	t.Equal(data.Z, "one")
	t.Equal(data.DataB.Z, "two")
	t.Equal(data.DataC.Z, "one")
}
