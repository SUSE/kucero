/*
Copyright (c) 2020 SUSE LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package node

import (
	"reflect"
	"testing"
	"time"
)

type stubClock struct {
}

func newStubClock() Clock {
	return &stubClock{}
}

func (s *stubClock) Now() time.Time {
	return time.Date(2000, time.January, 02, 03, 04, 05, 06, time.UTC)
}

func Test_checkExpiry(t *testing.T) {
	stubClock := newStubClock()

	tests := []struct {
		name                    string
		inputT                  time.Time
		inputExpiryTimeToRotate time.Duration
		expect                  bool
	}{
		{
			name:                    "expired certificate",
			inputT:                  time.Date(1960, time.May, 12, 02, 29, 00, 00, time.UTC),
			inputExpiryTimeToRotate: time.Minute,
			expect:                  true,
		},
		{
			name:                    "going to expire certificate",
			inputT:                  time.Date(2000, time.January, 02, 03, 05, 05, 06, time.UTC),
			inputExpiryTimeToRotate: time.Minute,
			expect:                  true,
		},
		{
			name:                    "still valid certificate",
			inputT:                  time.Date(2100, time.May, 12, 02, 29, 00, 00, time.UTC),
			inputExpiryTimeToRotate: time.Minute,
			expect:                  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := checkCertificateExpiry(tt.name, tt.inputT, tt.inputExpiryTimeToRotate, stubClock)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("got %t is not equals to expected %t", got, tt.expect)
			}
		})
	}
}
