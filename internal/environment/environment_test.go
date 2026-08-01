package environment

import (
	"testing"
	"time"
)

func TestEnvironmentMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		env  *Environment
	}{
		{
			name: "with variables",
			env:  &Environment{Name: "dev", Variables: map[string]string{"BASE_URL": "http://localhost:8080", "TOKEN": "abc"}, Created: time.Now(), Updated: time.Now()},
		},
		{
			name: "empty variables",
			env:  &Environment{Name: "prod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.env.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON failed: %v", err)
			}
			decoded, err := UnmarshalJSON(encoded)
			if err != nil {
				t.Fatalf("UnmarshalJSON failed: %v", err)
			}
			if decoded.Name != tt.env.Name {
				t.Errorf("name mismatch: got %q want %q", decoded.Name, tt.env.Name)
			}
			if len(decoded.Variables) != len(tt.env.Variables) {
				t.Errorf("variables length mismatch: got %d want %d", len(decoded.Variables), len(tt.env.Variables))
			}
			for k, v := range tt.env.Variables {
				if decoded.Variables[k] != v {
					t.Errorf("variable %q mismatch: got %q want %q", k, decoded.Variables[k], v)
				}
			}
		})
	}
}

func TestUnmarshalJSONInvalid(t *testing.T) {
	if _, err := UnmarshalJSON("not json"); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
