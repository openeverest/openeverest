// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testdata

// +openapi:export=User
// User represents a user in the system.
type User struct {
	// Username is the unique identifier for the user.
	Username string `json:"username"`
	// Email is the contact email for the user.
	Email string `json:"email"`
	// Age is the user's age in years.
	Age int `json:"age"`
	// IsAdmin indicates if the user has admin privileges.
	IsAdmin bool `json:"isAdmin"`
}

// +openapi:export=Config
// Config is the configuration object.
type Config struct {
	// Timeout specifies the timeout duration in seconds.
	Timeout int32 `json:"timeout"`
	// Retries is the number of retry attempts.
	Retries int64 `json:"retries"`
	// Metadata stores additional configuration data.
	Metadata map[string]string `json:"metadata"`
	// Tags are custom tags (omitted from schema as per JSON tag).
	Tags map[string]interface{} `json:"-"`
}
