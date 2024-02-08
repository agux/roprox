package ua

import (
	"net/http"
	"testing"

	"github.com/agux/roprox/conf"
)

func TestGetJSON(t *testing.T) {
	// Define your test cases
	tests := []struct {
		name         string
		mockResponse string
		mockStatus   int
		wantError    bool
	}{
		{
			name:       "Successful request",
			mockStatus: http.StatusOK,
			wantError:  false,
		},
		{
			name:      "Network error",
			wantError: true,
		},
		// Add more test cases if needed
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Use the URL of the mock server unless we're testing network error.
			url := conf.Args.DataSource.UserAgents
			if tc.name == "Network error" {
				url = "http://localhost:0"
			}

			uam := userAgentsMe{}
			jsonStr, err := uam.getJSON(url)

			if tc.wantError {
				if err == nil {
					t.Errorf("Expected an error but didn't get one")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				// if jsonStr is empty, then t.Errorf
				if jsonStr == "" {
					t.Errorf("Expected non-empty JSON string but got an empty string")
				}
			}
		})
	}
}

func Test_userAgentsMe_get(t *testing.T) {
	tests := []struct {
		name    string
		uam     userAgentsMe
		wantErr bool
	}{
		{
			name: "litmus",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uam := userAgentsMe{}
			gotAgents, err := uam.get()
			if (err != nil) != tt.wantErr {
				t.Errorf("userAgentsMe.get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// if gotAgents is empty, test fail
			if len(gotAgents) == 0 {
				t.Errorf("Expected non-empty list of user agents but got an empty list")
				return
			}

			// loop through the gotAgents array. if any of the UserAgent is valid but the string value is empty, test fail.
			for _, agent := range gotAgents {
				if agent.UserAgent.Valid && agent.UserAgent.String == "" {
					t.Errorf("Found valid UserAgent with empty string value")
					return
				}
			}
		})
	}
}
