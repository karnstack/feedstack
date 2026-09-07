package feed

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"
)

func TestDecodeLegacy(t *testing.T) {
	tests := []struct {
		name    string
		doc     string
		want    []string // expected titles, in order
		wantErr bool
	}{
		{
			name: "tagged entry gets a label",
			doc:  `{"channel":{"title":"morning brew","items":[{"headline":"go 1.26 ships","url":"https://brew.example/1","tags":["go"]}]}}`,
			want: []string{"[go] go 1.26 ships"},
		},
		{
			name: "tagless entry ships plain",
			doc:  `{"channel":{"title":"morning brew","items":[{"headline":"the archive post","url":"https://brew.example/2"}]}}`,
			want: []string{"the archive post"},
		},
		{
			name: "empty tags array ships plain",
			doc:  `{"channel":{"title":"morning brew","items":[{"headline":"quiet week","url":"https://brew.example/3","tags":[]}]}}`,
			want: []string{"quiet week"},
		},
		{
			name: "numeric tag ships plain",
			doc:  `{"channel":{"title":"morning brew","items":[{"headline":"issue 101","url":"https://brew.example/4","tags":[101]}]}}`,
			want: []string{"issue 101"},
		},
		{
			name: "entry missing headline is skipped",
			doc:  `{"channel":{"title":"morning brew","items":[{"url":"https://brew.example/5"}]}}`,
			want: nil,
		},
		{
			name:    "missing channel is an error",
			doc:     `{"exporter":"brewdump 0.4","items":[]}`,
			wantErr: true,
		},
		{
			name:    "items not an array is an error",
			doc:     `{"channel":{"title":"morning brew","items":"oops"}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc map[string]any
			if err := json.Unmarshal([]byte(tt.doc), &doc); err != nil {
				t.Fatalf("test setup: bad JSON in doc: %v", err)
			}

			items, err := decodeLegacy(doc)
			if (err != nil) != tt.wantErr {
				t.Fatalf("decodeLegacy() error = %v, wantErr %v", err, tt.wantErr)
			}

			var got []string
			for _, item := range items {
				got = append(got, item.Title)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("decodeLegacy() titles = %q, want %q", got, tt.want)
			}
		})
	}
}

func FuzzDecodeItems(f *testing.F) {
	f.Add([]byte(`{"version":"https://jsonfeed.org/version/1.1","title":"seed feed","items":[{"title":"a","url":"https://feeds.example/a"}]}`))
	f.Add([]byte(`{"exporter":"brewdump 0.4","channel":{"title":"seed","items":[{"headline":"a","url":"https://feeds.example/b","tags":["go"]}]}}`))
	f.Add([]byte(`not json at all`))

	f.Fuzz(func(t *testing.T, data []byte) {
		// The property is survival: no input may panic the parser.
		// An error return is a correct answer; rejecting garbage is the job.
		_, _ = Decode(bytes.NewReader(data))
	})
}
