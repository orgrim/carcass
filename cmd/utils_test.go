// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"os/user"
	"path/filepath"
	"testing"
)

func TestExpandDataDir(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Errorf("could not get current user: %s", err)
	}

	var tests = []struct {
		input string
		want  string
	}{
		{"", "."},
		{"/truc/truc/../muche", "/truc/muche"},
		{"./truc/muche", "truc/muche"},
		{"~/truc/muche/dir", filepath.Clean(filepath.Join(u.HomeDir, "/truc/muche/dir"))},
		{fmt.Sprintf("~%s/truc/muche", u.Username), filepath.Clean(filepath.Join(u.HomeDir, "/truc/muche"))},
	}

	for i, st := range tests {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			got, err := expandDataDir(st.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if got != st.want {
				t.Errorf("got: %v, want %v", got, st.want)
			}
		})
	}
}

func TestHasForbiddenChars(t *testing.T) {
	var tests = []struct {
		input string
		want  bool
	}{
		{"-a.b0Z", false},
		{"aa/bb", true},
		{"../../../?oups", true},
		{"deb.local", false},
	}

	for i, st := range tests {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			got := hasForbiddenChars(st.input)
			if got != st.want {
				t.Errorf("got: %v, want %v", got, st.want)
			}
		})
	}
}
