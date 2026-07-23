package cli

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantName  string
		wantToken string
		wantErr   bool
	}{
		{
			name:      "valid connect command",
			args:      []string{"connect", "my-secret-token"},
			wantName:  "connect",
			wantToken: "my-secret-token",
			wantErr:   false,
		},
		{
			name:    "no args",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "connect missing token",
			args:    []string{"connect"},
			wantErr: true,
		},
		{
			name:    "connect empty token",
			args:    []string{"connect", ""},
			wantErr: true,
		},
		{
			name:    "unknown command",
			args:    []string{"disconnect"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := Parse(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if cmd.Name != tt.wantName {
					t.Errorf("Parse() Name = %v, want %v", cmd.Name, tt.wantName)
				}
				if cmd.Token != tt.wantToken {
					t.Errorf("Parse() Token = %v, want %v", cmd.Token, tt.wantToken)
				}
			}
		})
	}
}
