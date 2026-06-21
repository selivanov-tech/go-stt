package config

import "testing"

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		profile Profile
		store   string
		wantErr bool
	}{
		{"local + memory", ProfileLocal, "memory", false},
		{"local + empty defaults ok", ProfileLocal, "", false},
		{"production + memory refused", ProfileProduction, "memory", true},
		{"production + empty refused", ProfileProduction, "", true},
		{"production + postgres ok", ProfileProduction, "postgres", false},
		{"production + sqlite ok", ProfileProduction, "sqlite", false},
		{"unknown profile refused", Profile("staging"), "postgres", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := Config{Profile: tc.profile, Store: tc.store}
			if err := c.Validate(); (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}
