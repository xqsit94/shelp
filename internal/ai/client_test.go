package ai

import (
	"reflect"
	"testing"
)

func TestParseCommands(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
		wantErr bool
	}{
		{
			name:    "plain array",
			content: `["ls -la", "pwd"]`,
			want:    []string{"ls -la", "pwd"},
		},
		{
			name:    "json fenced",
			content: "```json\n[\"ls -la\"]\n```",
			want:    []string{"ls -la"},
		},
		{
			name:    "bare fenced",
			content: "```\n[\"ls -la\"]\n```",
			want:    []string{"ls -la"},
		},
		{
			name:    "surrounding whitespace",
			content: "  \n\t[\"ls -la\"]\n  ",
			want:    []string{"ls -la"},
		},
		{
			name:    "empty array",
			content: "[]",
			want:    []string{},
		},
		{
			name:    "invalid json",
			content: "not json at all",
			wantErr: true,
		},
		{
			name:    "object instead of array",
			content: `{"command": "ls"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCommands(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseCommands(%q) = %v, want error", tt.content, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCommands(%q) returned error: %v", tt.content, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseCommands(%q) = %#v, want %#v", tt.content, got, tt.want)
			}
		})
	}
}
