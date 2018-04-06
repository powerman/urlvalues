package urlvalues

import (
	"net/url"
	"testing"

	"github.com/powerman/check"
)

func TestUsage(tt *testing.T) {
	t := check.T(tt)
	var v struct{}
	var i int
	d := NewStrictDecoder(NewDecoderOpts())
	t.PanicMatch(func() { d.Unmarshal(nil, &v) }, `^data .* nil`)
	t.PanicMatch(func() { d.Unmarshal(url.Values{}, nil) }, `^v .* non-nil`)
	t.PanicMatch(func() { d.Unmarshal(url.Values{}, v) }, `^v .* pointer`)
	t.PanicMatch(func() { d.Unmarshal(url.Values{}, &i) }, `^v .* struct`)
}

func TestEmpty(tt *testing.T) {
	t := check.T(tt)
	var v1 struct{}
	var v2 struct{ S string }
	d := NewStrictDecoder(NewDecoderOpts())
	t.Nil(d.Unmarshal(url.Values{}, &v1))
	t.Nil(d.Unmarshal(url.Values{}, &v2))
	t.Zero(v2)
}

func TestSmoke(tt *testing.T) {
	t := check.T(tt)
	var v = struct {
		S string `form:"s" validate:"eq=good"`
	}{S: "neutral"}
	d := NewStrictDecoder(NewDecoderOpts())
	t.DeepEqual(d.Unmarshal(url.Values{"s": {"bad"}}, &v), Errs{url.Values{
		"S": {"failed validation: eq=good"},
	}})
	t.Equal(v.S, "bad")
	t.Nil(d.Unmarshal(url.Values{"s": {"good"}}, &v))
	t.Equal(v.S, "good")
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
	d := NewStrictDecoder(NewDecoderOpts())
	t.Nil(d.Unmarshal(url.Values{
		// "AI[1]":            {"42"},
		// "AF[0].I":          {"42"},
		"SI[9999]":        {"42"},
		"SF[0000].I":      {"42"},
		"SSI[9999][9999]": {"42"},
		"SSI[0][0]":       {"42"},
		// "ASAI[9][9999][1]": {"42"},
		// "ASAI[0][0][0]":    {"42"},
	}, &data))
	t.DeepEqual(d.Unmarshal(url.Values{
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
	}, &data), Errs{url.Values{
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
	d := NewStrictDecoder(NewDecoderOpts())
	t.Nil(d.Unmarshal(url.Values{
		"S1":      {"10", "20"},
		"S2[0]":   {"10"},
		"S2[1]":   {"20"},
		"SF[0].I": {"42"},
		"I":       {"42"},
		"MI[a]":   {"42"},
		"MI[b]":   {"42"},
		"MSI[a]":  {"10", "20"},
		"MSI[b]":  {"30", "40"},
	}, &data))
	t.DeepEqual(d.Unmarshal(url.Values{
		"S2[0]":   {"10", "20"},
		"S2[1]":   {"30", "40"},
		"SF[0].I": {"42", "43"},
		"I":       {"42", "43"},
		"MI[a]":   {"10", "20"},
		"MI[b]":   {"10", "20"},
	}, &data), Errs{url.Values{
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
	d := NewStrictDecoder(NewDecoderOpts())
	t.Nil(d.Unmarshal(url.Values{
	// "SAI[42]": {"10", "20"},
	// "AI": {"10", "20"},
	}, &data))
	t.DeepEqual(d.Unmarshal(url.Values{
		"SAI[0]":  {"10", "20", "30"},
		"SAI[42]": {"10", "20", "30"},
		"AI":      {"10", "20", "30"},
	}, &data), Errs{url.Values{
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
		SI []int
		Embed
	}
	d := NewStrictDecoder(NewDecoderOpts())
	t.Nil(d.Unmarshal(url.Values{
		"SI": {"10", "20"},
		"I":  {"42"},
	}, &data))
	t.Nil(d.Unmarshal(url.Values{
		"SI[0]":   {"10"},
		"SI[1]":   {"20"},
		"Embed.I": {"42"},
	}, &data))
	t.DeepEqual(d.Unmarshal(url.Values{
		"SI":      {"10"},
		"SI[1]":   {"20"},
		"I":       {"30"},
		"Embed.I": {"40"},
	}, &data), Errs{url.Values{
		"SI": {"multiple names for same value"},
		"I":  {"multiple names for same value"},
	}})
}

func TestRequired(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		I int    `form:"i,required"`
		S string `form:",required"`
	}
	d := NewStrictDecoder(NewDecoderOpts())
	t.Nil(d.Unmarshal(url.Values{
		"i": {"0"},
		"S": {""},
	}, &data))
	t.DeepEqual(d.Unmarshal(url.Values{}, &data), Errs{url.Values{
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
	d := NewStrictDecoder(NewDecoderOpts())
	t.Match(d.Unmarshal(url.Values{
		"A": {"one"},
	}, &data), `unknown name: "A"`)
	t.Match(d.Unmarshal(url.Values{
		"M": {"one"},
	}, &data), `unknown name: "M"`)
	t.Match(d.Unmarshal(url.Values{
		"F": {"42"},
	}, &data), `unknown name: "F"`)
}

func BenchmarkSmallFailure(b *testing.B) {
	var data struct {
		FName string `form:",required"`
		LName string `form:",required"`
		Age   uint   `validate:"max=130"`
	}
	d := NewStrictDecoder(NewDecoderOpts())
	d.Unmarshal(url.Values{}, &data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if nil == d.Unmarshal(url.Values{
			"Age": {"42"},
		}, &data) {
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
	d := NewStrictDecoder(NewDecoderOpts())
	d.Unmarshal(url.Values{}, &data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if nil != d.Unmarshal(url.Values{
			"FName": {"First"},
			"LName": {"Last"},
			"Age":   {"42"},
		}, &data) {
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
	d := NewStrictDecoder(NewDecoderOpts())
	d.Unmarshal(url.Values{}, &data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if nil != d.decode(&data, url.Values{
			"FName": {"First"},
			"LName": {"Last"},
			"Age":   {"42"},
		}) {
			b.FailNow()
		}
		if nil != d.validate.Struct(&data) {
			b.FailNow()
		}
	}
}

func BenchmarkLargeFailure(b *testing.B) {
	var data DataA
	d := NewStrictDecoder(NewDecoderOpts())
	d.Unmarshal(url.Values{}, &data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if nil == d.Unmarshal(url.Values{
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
		}, &data) {
			b.FailNow()
		}
	}
}

func BenchmarkLargeSuccess(b *testing.B) {
	var data DataA
	d := NewStrictDecoder(NewDecoderOpts())
	d.Unmarshal(url.Values{}, &data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if nil != d.Unmarshal(url.Values{
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
		}, &data) {
			b.FailNow()
		}
	}
}

func BenchmarkLargeSuccessLoose(b *testing.B) {
	var data DataA
	d := NewStrictDecoder(NewDecoderOpts())
	d.Unmarshal(url.Values{}, &data)
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
		if nil != d.validate.Struct(&data) {
			b.FailNow()
		}
	}
}
