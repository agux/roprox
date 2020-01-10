package util

import (
	"math/rand"
	"testing"
	"time"
)

func TestPickUserAgent(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	tests := []struct {
		name    string
		wantUa  string
		wantErr bool
	}{
		{
			name:    "default",
			wantUa:  "any",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUa, err := PickUserAgent()
			if (err != nil) != tt.wantErr {
				t.Errorf("PickUserAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotUa != tt.wantUa {
				t.Errorf("PickUserAgent() = %v, want %v", gotUa, tt.wantUa)
			}
		})
	}
}
