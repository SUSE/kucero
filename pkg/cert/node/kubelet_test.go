package node

import (
	"reflect"
	"testing"
	"time"
)

func Test_parseKubeletCheckExpiration(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		expect    time.Time
	}{
		{
			name:   "normal case",
			input:  "notAfter=Jan 02 15:04:05 2006 UTC",
			expect: time.Date(2006, time.January, 02, 15, 04, 05, 00, time.UTC),
		},
		{
			name:      "without notAfter key",
			input:     "Jan 02 15:04:05 2006 UTC",
			expectErr: true,
		},
		{
			name:      "without equal sign",
			input:     "notAfter",
			expectErr: true,
		},
		{
			name:      "time format incorrect",
			input:     "notAfter=Jan 02 15:04:05 2006",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			gotT, err := parseKubeletCheckExpiration(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Error("Expect error but got no error")
				}
				return
			}

			if !reflect.DeepEqual(*gotT, tt.expect) {
				t.Errorf("gotT %v is not equals to expected %v", *gotT, tt.expect)
			}
		})
	}
}
