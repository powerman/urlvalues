package urlvalues_test

import (
	"net/url"
	"testing"

	"github.com/go-playground/form"
	"github.com/powerman/check"
)

func TestDecodeMixedIndex(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		A [5]int
		S []int
	}
	for i := 0; i < 10; i++ { // ensure stable keys order
		data.A = [5]int{}
		data.S = []int{}
		form.NewDecoder().Decode(&data, url.Values{
			"A":    {"10", "20", "30"},
			"A[1]": {"200"},
			"A[4]": {"400"},
			"S":    {"10", "20", "30"},
			"S[1]": {"200"},
			"S[4]": {"400"},
		})
		t.DeepEqual(data.A, [5]int{10, 200, 30, 0, 400})
		t.DeepEqual(data.S, []int{10, 200, 30, 0, 400})
	}
}

func TestDecodeArrayBug29(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		A [2]string
	}
	form.NewDecoder().Decode(&data, url.Values{
		"A":    {"10"},
		"A[1]": {"20"},
	})
	t.DeepEqual(data.A, [2]string{"10", "20"})
}

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
		"A[2]": {"20"},
	})
	t.DeepEqual(data.A, []string{"10", "", "20"})
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

func TestDecodeArrayBug33(tt *testing.T) {
	t := check.T(tt)
	var data struct {
		A [3]string
	}
	cases := []struct {
		values url.Values
		want   [3]string
	}{
		{url.Values{"A": {"10"}},
			[3]string{"10", "", ""}},
		{url.Values{"A": {"10", "20"}},
			[3]string{"10", "20", ""}},
		{url.Values{"A[1]": {"20"}},
			[3]string{"", "20", ""}},
		{url.Values{"A": {"10"}, "A[2]": {"30"}},
			[3]string{"10", "", "30"}},
	}
	for _, v := range cases {
		data.A = [3]string{}
		form.NewDecoder().Decode(&data, v.values)
		t.DeepEqual(data.A, v.want)
	}
}
