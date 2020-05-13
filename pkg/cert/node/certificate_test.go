package node

import (
	"reflect"
	"testing"
	"time"
)

func Test_checkExpiry(t *testing.T) {
	tests := []struct {
		name                    string
		inputT                  time.Time
		inputExpiryTimeToRotate time.Duration
		expect                  bool
	}{
		{
			name:                    "cert1",
			inputT:                  time.Date(1960, time.May, 12, 02, 29, 00, 00, time.UTC),
			inputExpiryTimeToRotate: time.Minute,
			expect:                  true,
		},
		{
			name:                    "cert2",
			inputT:                  time.Date(2100, time.May, 12, 02, 29, 00, 00, time.UTC),
			inputExpiryTimeToRotate: time.Minute,
			expect:                  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := checkExpiry(tt.name, tt.inputT, tt.inputExpiryTimeToRotate)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("got %t is not equals to expected %t", got, tt.expect)
			}
		})
	}
}
