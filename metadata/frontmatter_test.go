package metadata_test

import (
	"testing"

	"github.com/dkotik/cuebook/metadata"
)

func TestFrontmatterGet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		zeroValue bool
		metadata  any
		want      any
	}{
		{
			name:      "zero value",
			zeroValue: true,
		},
		{
			name: "missing metadata",
		},
		{
			name:     "metadata value",
			metadata: "value",
			want:     "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var frontmatter metadata.Frontmatter
			if !tt.zeroValue {
				frontmatter = metadata.NewFrontmatter([]byte("// A title\\n"), metadata.NewParser())
				if tt.metadata != nil {
					frontmatter.Node.OwnerDocument().AddMeta("field", tt.metadata)
				}
			}
			if got := frontmatter.Get("field"); got != tt.want {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFrontmatterTitleAndDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      string
		wantTitle   string
		wantDetails string
	}{
		{
			name: "empty source",
		},
		{
			name:      "title only",
			source:    "// A title\npackage sample\n",
			wantTitle: "A title",
		},
		{
			name:        "title and multi-paragraph description",
			source:      "// A title\n//\n// First paragraph\n// continues here\n//\n// Second paragraph\npackage sample\n",
			wantTitle:   "A title",
			wantDetails: "First paragraph\ncontinues here\nSecond paragraph\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			frontmatter := metadata.NewFrontmatter([]byte(tt.source), metadata.NewParser())
			if got := frontmatter.Title(); got != tt.wantTitle {
				t.Errorf("Title() = %q, want %q", got, tt.wantTitle)
			}
			if got := frontmatter.Description(); got != tt.wantDetails {
				t.Errorf("Description() = %q, want %q", got, tt.wantDetails)
			}
		})
	}
}
