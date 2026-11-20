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

export interface TlsConfigFormValues {
  insecureSkipVerify: boolean;
  caSecretName: string;
  caSecretKey: string;
}

// How an object trusts the server it reaches: the CA bundle in a Secret's
// key, or no verification at all.
export interface TlsConfig {
  ca?: { secretKeyRef?: { name: string; key: string } };
  insecureSkipVerify?: boolean;
}
