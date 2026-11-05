package metadata

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

func TestIsOpen(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		source string // the tested value is at the path "value"
		want   bool
	}{
		{name: "plain struct", source: `value: {Name: "x"}`, want: true},
		{name: "closed struct", source: `value: close({Name: "x"})`, want: false},
		{name: "closed struct with ellipsis", source: `value: close({Name: "x", ...})`, want: true},
		{name: "definition", source: `#entry: {Name: string}, value: #entry`, want: false},
		{name: "definition with ellipsis", source: `#entry: {Name: string, ...}, value: #entry`, want: true},
		{name: "definition with ellipsis after comment", source: "#entry: {\n\tName: string\n\t... // allow any additional fields\n}\nvalue: #entry", want: true},
		{name: "list entry of closed definition", source: `#entry: {Name: string}, value: ([...#entry] & [{Name: "x"}])[0]`, want: false},
		{name: "list entry of open definition", source: `#entry: {Name: string, ...}, value: ([...#entry] & [{Name: "x"}])[0]`, want: true},
		{name: "nested closed struct", source: `#entry: {Address: {City: string}}, value: #entry.Address`, want: false},
		{name: "nested open struct", source: `#entry: {Address: {City: string, ...}}, value: #entry.Address`, want: true},
		{name: "partial pattern constraint", source: `#entry: {Name: string, [=~"^x"]: string}, value: #entry`, want: false},
		{name: "any disjunct is open", source: `#entry: {Name: string} | {Other: string, ...}, value: #entry`, want: true},
		{name: "open list", source: `value: [...int]`, want: true},
		{name: "open list with prefix", source: `#list: [int, ...], value: #list`, want: true},
		{name: "closed list", source: `value: [1, 2]`, want: false},
		{name: "string", source: `value: "x"`, want: false},
		{name: "string type", source: `value: string`, want: false},
		{name: "top", source: `value: _`, want: false},
		{name: "missing value", source: `other: 1`, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			document := cuecontext.New().CompileString(tt.source)
			if err := document.Err(); err != nil {
				t.Fatal(err)
			}
			if got := IsOpen(document.LookupPath(cue.ParsePath("value"))); got != tt.want {
				t.Errorf("IsOpen() = %t, want %t", got, tt.want)
			}
		})
	}
}
