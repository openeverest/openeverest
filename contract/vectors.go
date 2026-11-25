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

package contract

import _ "embed"

// OrderVectors is the JSON document of the cases every implementation of the
// check must pass, so that a Helm template, the UI or a CEL rule decides
// exactly as Check, IsDevelopment and AcknowledgementHolds do:
//
//   - "order" lists valid versions in strictly ascending precedence;
//   - "invalid" lists values Valid rejects;
//   - "development" lists objects with the fields "version", the argument
//     of IsDevelopment, and "development", its result;
//   - "check" lists objects with the fields "ext", "min" and "cur", the
//     arguments of Check, and "status" and "reason", its Result;
//   - "acknowledge" lists objects with the fields "ack", "contractVersion"
//     and "coreVersion", the arguments of AcknowledgementHolds, and "holds",
//     its result.
//
//go:embed testdata/order.json
var OrderVectors []byte
