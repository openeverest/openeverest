// everest
// Copyright (C) 2023 Percona LLC
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

// Package hack contains code generation directives for the project.
package hack

//go:generate go tool oapi-codegen --config=crds.cfg.yml  ../api/openapi/crds.gen.yaml
//go:generate go tool oapi-codegen --config=server.cfg.yml  ../api/openapi/http-api.yaml
//go:generate go tool oapi-codegen --config=client-crds.cfg.yml  ../api/openapi/crds.gen.yaml
//go:generate go tool oapi-codegen --config=client.cfg.yml  ../api/openapi/http-api.yaml
