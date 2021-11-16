package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadOptionsFromConfig(t *testing.T) {
	tt := []struct {
		test        string
		configPath  string
		expected    *OpenerOptions
		expectedErr string
	}{
		{
			"unix",
			filepath.Join("testdata", "config", "unix.yaml"),
			&OpenerOptions{
				Network: "unix",
				Address: "~/.opener.sock",
			},
			"",
		},
		{
			"tcp",
			filepath.Join("testdata", "config", "tcp.yaml"),
			&OpenerOptions{
				Network: "tcp",
				Address: "127.0.0.1:9000",
			},
			"",
		},
		{
			"empty",
			filepath.Join("testdata", "config", "empty.yaml"),
			&OpenerOptions{},
			"",
		},
		{
			"no such file",
			filepath.Join("testdata", "config", "no-such-file.yaml"),
			&OpenerOptions{},
			"stat testdata/config/no-such-file.yaml: no such file or directory",
		},
	}

	for _, tc := range tt {
		t.Run(tc.test, func(t *testing.T) {
			o := &OpenerOptions{}
			err := LoadOpenerOptionsFromConfig(tc.configPath, o)
			if err == nil {
				if tc.expectedErr != "" {
					t.Errorf("expected err nil, but %q", err)
				}
			} else {
				if tc.expectedErr != err.Error() {
					t.Errorf("expected err %q, but %q", tc.expectedErr, err)
				}
			}

			if !reflect.DeepEqual(tc.expected, o) {
				t.Errorf("expected %#v, but %#v", tc.expected, o)
			}
		})
	}
}
