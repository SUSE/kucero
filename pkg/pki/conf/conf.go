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

package conf

type Config interface {
	// CheckConfig checks whether the configuration need to be update
	// returns the configuration and it's update config callback function
	CheckConfig() ([]string, error)

	// UpdateConfig updates the configuration by
	// passing the configuration and it's update config callback function
	UpdateConfig([]string) error
}
