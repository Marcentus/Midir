package main

import "testing"

func TestStartupPromiscuousMode(t *testing.T) {
	cases := []struct {
		name string
		nic  string
		file string
		cfg  *CaptureConfig
		want bool
	}{
		{"same PC default no settings", "\\Device\\NPF_test", "", nil, false},
		{"same PC saved off", "nic0", "", &CaptureConfig{Promiscuous: false}, false},
		{"mirror NIC saved on", "nic0", "", &CaptureConfig{Promiscuous: true}, true},
		{"offline playback saved on", "", "session.pcapng", &CaptureConfig{Promiscuous: true}, false},
		{"offline playback default", "", "session.pcapng", nil, false},
		{"no capture", "", "", &CaptureConfig{Promiscuous: true}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := startupPromiscuousMode(tc.nic, tc.file, tc.cfg)
			if got != tc.want {
				t.Fatalf("startupPromiscuousMode(%q,%q,%v)=%v, want %v", tc.nic, tc.file, tc.cfg, got, tc.want)
			}
		})
	}
}
