package urlvalues

import (
	"net/url"
	"sort"
	"testing"

	"github.com/powerman/check"
)

func TestUsage(tt *testing.T) {
	t := check.T(tt)
	var v struct{}
	var i int
	d := NewStrictDecoder()
	t.PanicMatch(func() { d.Decode(&v, nil) }, `^data .* nil`)
	t.PanicMatch(func() { d.Decode(nil, url.Values{}) }, `^v .* non-nil`)
	t.PanicMatch(func() { d.Decode(v, url.Values{}) }, `^v .* pointer`)
	t.PanicMatch(func() { d.Decode(&i, url.Values{}) }, `^v .* struct`)
}

func TestBadTagForm(tt *testing.T) {
	t := check.T(tt)
	var v1 struct {
		S string `form:",wrong"`
	}
	var v2 struct {
		S string `form:"s,omitempty,required,wrong"`
	}
	d := NewStrictDecoder()
	t.PanicMatch(func() { d.Decode(&v1, url.Values{}) }, `"wrong" .* "S"`)
	t.PanicMatch(func() { d.Decode(&v2, url.Values{"s": {""}}) }, `"wrong" .* "S"`)
}

func TestEmpty(tt *testing.T) {
	t := check.T(tt)
	var v1 struct{}
	var v2 struct{ S string }
	d := NewStrictDecoder()
	t.Nil(d.Decode(&v1, url.Values{}))
	t.Nil(d.Decode(&v2, url.Values{}))
	t.Zero(v2)
}

func TestSmoke(tt *testing.T) {
	t := check.T(tt)
	var v = struct {
		I int `form:"i"`
	}{I: 42}
	d := NewStrictDecoder()
	t.DeepEqual(d.Decode(&v, url.Values{"i": {"bad"}}), Errs{url.Values{
		"i": {"wrong type"},
	}})
	t.Equal(v.I, 42)
	t.Nil(d.Decode(&v, url.Values{"i": {"10"}}))
	t.Equal(v.I, 10)
}

func TestIndexOutOfBounds(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		AI   [2]int
		AF   [2]struct{ I int }
		SI   []int
		SF   []struct{ I int }
		SSI  [][]int
		ASAI [10][][2]int
	}
	d := NewStrictDecoder()
	t.Nil(d.Decode(&data, url.Values{
		"AI[1]":            {"42"},
		"AF[0].I":          {"42"},
		"SI[9999]":         {"42"},
		"SF[0000].I":       {"42"},
		"SSI[9999][9999]":  {"42"},
		"SSI[0][0]":        {"42"},
		"ASAI[9][9999][1]": {"42"},
		"ASAI[0][0][0]":    {"42"},
	}))
	t.DeepEqual(d.Decode(&data, url.Values{
		"AI[2]":             {"42"},
		"AF[2].I":           {"42"},
		"SI[10000]":         {"42"},
		"SF[10000].I":       {"42"},
		"SSI[10000][0]":     {"42"},
		"SSI[0][10000]":     {"42"},
		"SSI[10000][10000]": {"42"},
		"ASAI[10][42][0]":   {"42"},
		"ASAI[9][10000][0]": {"42"},
		"ASAI[9][42][2]":    {"42"},
	}), Errs{url.Values{
		"AI[idx]":   {"index out-of-bounds"},
		"AF[idx].I": {"index out-of-bounds"},
		"SI[idx]":   {"index out-of-bounds"},
		"SF[idx].I": {"index out-of-bounds"},
		"SSI[idx][idx]": {
			"index out-of-bounds",
			"index out-of-bounds",
			"index out-of-bounds",
			"index out-of-bounds",
		},
		"ASAI[idx][idx][idx]": {
			"index out-of-bounds",
			"index out-of-bounds",
			"index out-of-bounds",
		},
	}})
}

func TestMultipleValues(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		S1  []int
		S2  []int
		SF  []struct{ I int }
		I   int
		MI  map[string]int
		MSI map[string][]int
	}
	d := NewStrictDecoder()
	t.Nil(d.Decode(&data, url.Values{
		"S1":      {"10", "20"},
		"S2[0]":   {"10"},
		"S2[1]":   {"20"},
		"SF[0].I": {"42"},
		"I":       {"42"},
		"MI[a]":   {"42"},
		"MI[b]":   {"42"},
		"MSI[a]":  {"10", "20"},
		"MSI[b]":  {"30", "40"},
	}))
	t.DeepEqual(d.Decode(&data, url.Values{
		"S2[0]":   {"10", "20"},
		"S2[1]":   {"30", "40"},
		"SF[0].I": {"42", "43"},
		"I":       {"42", "43"},
		"MI[a]":   {"10", "20"},
		"MI[b]":   {"10", "20"},
	}), Errs{url.Values{
		"S2[idx]":   {"multiple values", "multiple values"},
		"SF[idx].I": {"multiple values"},
		"I":         {"multiple values"},
		"MI[key]":   {"multiple values", "multiple values"},
	}})
}

func TestTooManyValues(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		SAI *[]*[2]int
		AI  [2]int
	}
	d := NewStrictDecoder()
	t.Nil(d.Decode(&data, url.Values{
		"SAI[42]": {"10", "20"},
		"AI":      {"10", "20"},
	}))
	t.DeepEqual(d.Decode(&data, url.Values{
		"SAI[0]":  {"10", "20", "30"},
		"SAI[42]": {"10", "20", "30"},
		"AI":      {"10", "20", "30"},
	}), Errs{url.Values{
		"SAI[idx]": {"too many values", "too many values"},
		"AI":       {"too many values"},
	}})
}

func TestMultipleNames(tt *testing.T) {
	t := check.T(tt)
	type Embed struct {
		I int
	}
	var data struct {
		AI [3]int
		SI []int
		Embed
	}
	d := NewStrictDecoder()
	t.Nil(d.Decode(&data, url.Values{
		"AI": {"10", "20"},
		"SI": {"10", "20"},
		"I":  {"42"},
	}))
	t.Nil(d.Decode(&data, url.Values{
		"AI[0]":   {"10"},
		"AI[2]":   {"30"},
		"SI[0]":   {"10"},
		"SI[1]":   {"20"},
		"Embed.I": {"42"},
	}))
	t.DeepEqual(d.Decode(&data, url.Values{
		"AI":      {"10"},
		"AI[1]":   {"20"},
		"SI":      {"10"},
		"SI[1]":   {"20"},
		"I":       {"30"},
		"Embed.I": {"40"},
	}), Errs{url.Values{
		"AI": {"multiple names for same value"},
		"SI": {"multiple names for same value"},
		"I":  {"multiple names for same value"},
	}})
}

func TestRequired(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		I int    `form:"i,omitempty,required"`
		S string `form:",required"`
	}
	d := NewStrictDecoder()
	t.Nil(d.Decode(&data, url.Values{
		"i": {"0"},
		"S": {""},
	}))
	t.DeepEqual(d.Decode(&data, url.Values{}), Errs{url.Values{
		"i": {"required"},
		"S": {"required"},
	}})
}

func TestUnknown(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		S string
		M map[int]int
		F struct {
			I int
		}
	}

	d := NewStrictDecoder()
	t.DeepEqual(d.Decode(&data, url.Values{
		"A": {"one"},
		"S": {"one", "two"},
	}), Errs{url.Values{
		"-": {"A"},
		"S": {"multiple values"},
	}})
	t.DeepEqual(d.Decode(&data, url.Values{
		"A": {"one"},
	}), Errs{url.Values{
		"-": {"A"},
	}})
	t.DeepEqual(d.Decode(&data, url.Values{
		"M": {"one"},
	}), Errs{url.Values{
		"-": {"M"},
	}})
	errs := d.Decode(&data, url.Values{
		"A": {"one"},
		"F": {"42"},
	})
	sort.Strings(errs.(Errs).Values["-"])
	t.DeepEqual(errs, Errs{url.Values{
		"-": {"A", "F"},
	}})

	d = NewStrictDecoder(IgnoreUnknown())
	t.Nil(d.Decode(&data, url.Values{
		"F": {"42"},
	}))
}

func TestPartial(tt *testing.T) {
	t := check.T(tt)
	type Part struct {
		I int
	}
	type Data struct {
		First Part `form:"-"`
		I     int
		last  Part
	}
	v := Data{First: Part{I: 10}, I: 20, last: Part{I: 30}}
	d := NewStrictDecoder()

	t.Nil(d.Decode(&v.First, url.Values{"I": {"100"}}))
	t.DeepEqual(v, Data{First: Part{I: 100}, I: 20, last: Part{I: 30}})

	errs := d.Decode(&v, url.Values{
		"First.I": {"199"},
		"I":       {"200"},
		"last.I":  {"399"},
	})
	sort.Strings(errs.(Errs).Values["-"])
	t.DeepEqual(errs, Errs{url.Values{
		"-": {"First.I", "last.I"},
	}})
	t.Nil(d.Decode(&v, url.Values{"I": {"200"}}))
	t.DeepEqual(v, Data{First: Part{I: 100}, I: 200, last: Part{I: 30}})

	t.Nil(d.Decode(&v.last, url.Values{"I": {"300"}}))
	t.DeepEqual(v, Data{First: Part{I: 100}, I: 200, last: Part{I: 300}})
}

func TestSkipEmbedded(tt *testing.T) {
	t := check.T(tt)
	type Part struct {
		I int
	}
	type Data struct {
		Part `form:"-"`
	}
	var v Data
	d := NewStrictDecoder()

	t.Nil(d.Decode(&v, url.Values{"I": {"10"}})) // XXX Is it ok to silently drop value?
	t.DeepEqual(v, Data{Part: Part{I: 0}})
}

func BenchmarkSmallFailure(b *testing.B) {
	var data struct {
		FName string `form:",required"`
		LName string `form:",required"`
		Age   uint   `validate:"max=130"`
	}
	d := NewStrictDecoder()
	d.Decode(&data, url.Values{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if nil == d.Decode(&data, url.Values{
			"Age": {"42"},
		}) {
			b.FailNow()
		}
	}
}

func BenchmarkSmallSuccess(b *testing.B) {
	var data struct {
		FName string `form:",required"`
		LName string `form:",required"`
		Age   uint   `validate:"max=130"`
	}
	d := NewStrictDecoder()
	d.Decode(&data, url.Values{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if nil != d.Decode(&data, url.Values{
			"FName": {"First"},
			"LName": {"Last"},
			"Age":   {"42"},
		}) {
			b.FailNow()
		}
	}
}

func BenchmarkSmallSuccessLoose(b *testing.B) {
	var data struct {
		FName string `form:",required"`
		LName string `form:",required"`
		Age   uint   `validate:"max=130"`
	}
	d := NewStrictDecoder()
	d.Decode(&data, url.Values{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if nil != d.decode(&data, url.Values{
			"FName": {"First"},
			"LName": {"Last"},
			"Age":   {"42"},
		}) {
			b.FailNow()
		}
	}
}

func BenchmarkLargeFailure(b *testing.B) {
	var data DataA
	d := NewStrictDecoder()
	d.Decode(&data, url.Values{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if nil == d.Decode(&data, url.Values{
			"A":             {"a"},
			"Y.I":           {"42"},
			"Y.S":           {"y.s"},
			"Z":             {"z"},
			"M[10]":         {"20"},
			"M[30]":         {"40"},
			"S2[a][0].Z":    {"datab.s2[a][0].z"},
			"S2[a][1].Z":    {"datab.s2[a][1].z"},
			"S2[b][0].Z":    {"datab.s2[b][0].z"},
			"DataB.zz":      {"datab.zz"},
			"DataB.C":       {"datab.c", "datab.datac.c"},
			"DataB.DataC.Z": {"datab.datac.z"},
			"C":             {"datac.c", "c"},
			"DataC.Z":       {"datac.z"},
		}) {
			b.FailNow()
		}
	}
}

func BenchmarkLargeSuccess(b *testing.B) {
	var data DataA
	d := NewStrictDecoder()
	d.Decode(&data, url.Values{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if nil != d.Decode(&data, url.Values{
			"A":             {"a"},
			"Y.I":           {"42"},
			"Y.S":           {"y.s"},
			"Z":             {"z"},
			"B":             {"datab.b - required"},
			"M[10]":         {"20"},
			"M[30]":         {"40"},
			"S2[a][0].Z":    {"datab.s2[a][0].z"},
			"S2[a][1].Z":    {"datab.s2[a][1].z"},
			"S2[b][0].Z":    {"datab.s2[b][0].z"},
			"DataB.zz":      {"datab.zz"},
			"DataB.C":       {"datab.c", "datab.datac.c"},
			"DataB.DataC.Z": {"datab.datac.z"},
			"C":             {"datac.c", "c"},
			"DataC.Z":       {"datac.z"},
		}) {
			b.FailNow()
		}
	}
}

func BenchmarkLargeSuccessLoose(b *testing.B) {
	var data DataA
	d := NewStrictDecoder()
	d.Decode(&data, url.Values{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if nil != d.decode(&data, url.Values{
			"A":             {"a"},
			"Y.I":           {"42"},
			"Y.S":           {"y.s"},
			"Z":             {"z"},
			"B":             {"datab.b - required"},
			"M[10]":         {"20"},
			"M[30]":         {"40"},
			"S2[a][0].Z":    {"datab.s2[a][0].z"},
			"S2[a][1].Z":    {"datab.s2[a][1].z"},
			"S2[b][0].Z":    {"datab.s2[b][0].z"},
			"DataB.zz":      {"datab.zz"},
			"DataB.C":       {"datab.c", "datab.datac.c"},
			"DataB.DataC.Z": {"datab.datac.z"},
			"C":             {"datac.c", "c"},
			"DataC.Z":       {"datac.z"},
		}) {
			b.FailNow()
		}
	}
}
