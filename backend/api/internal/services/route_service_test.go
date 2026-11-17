package services

import "testing"

func TestNormalizeRoadType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "canonical uppercase", input: "HIGHWAY", want: "HIGHWAY"},
		{name: "lowercase", input: "highway", want: "HIGHWAY"},
		{name: "whitespace", input: " expressway ", want: "EXPRESSWAY"},
		{name: "legacy mixed", input: "mixed", want: "UNKNOWN"},
		{name: "legacy suburban", input: "Suburban", want: "UNKNOWN"},
		{name: "blank fallback", input: "  ", want: "UNKNOWN"},
		{name: "invalid value", input: "motorway", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeRoadType(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("normalizeRoadType(%q) error = %v, wantErr %v", test.input, err, test.wantErr)
			}
			if err == nil && got != test.want {
				t.Errorf("normalizeRoadType(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}