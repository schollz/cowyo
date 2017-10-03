package kebabcase_test

import (
	"fmt"
	"reflect"
	"testing"

	"dmitri.shuralyov.com/kebabcase"
	"github.com/shurcooL/graphql/ident"
)

func Example_kebabCaseToMixedCaps() {
	fmt.Println(kebabcase.Parse("client-mutation-id").ToMixedCaps())

	// Output: ClientMutationID
}

func TestParse(t *testing.T) {
	tests := []struct {
		in   string
		want ident.Name
	}{
		{in: "book", want: ident.Name{"book"}},
		{in: "bookmark", want: ident.Name{"bookmark"}},
		{in: "arrow-right", want: ident.Name{"arrow", "right"}},
		{in: "arrow-small-right", want: ident.Name{"arrow", "small", "right"}},
		{in: "device-camera-video-audio", want: ident.Name{"device", "camera", "video", "audio"}},
		{in: "rss", want: ident.Name{"rss"}},
	}
	for _, tc := range tests {
		got := kebabcase.Parse(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("got: %q, want: %q", got, tc.want)
		}
	}
}

func BenchmarkParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		kebabcase.Parse("book")
		kebabcase.Parse("bookmark")
		kebabcase.Parse("arrow-right")
		kebabcase.Parse("arrow-small-right")
		kebabcase.Parse("device-camera-video-audio")
		kebabcase.Parse("rss")
	}
}
