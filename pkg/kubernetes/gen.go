// everest
// Copyright (C) 2025 Percona LLC
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

package kubernetes

//go:generate go tool ifacemaker -f accounts.go -f backup_storage.go -f olm_catalog_source.go -f configmap.go -f olm_cluster_service_version.go -f crd.go -f database_cluster.go -f database_cluster_backup.go -f database_cluster_restore.go -f database_engine.go -f data_importer.go -f data_import_job.go -f deployment.go -f olm_install_plan.go -f kubernetes.go -f monitoring_config.go -f namespace.go -f object.go -f operator.go -f jwt.go -f oidc.go -f pod_scheduling_policy.go -f load_balancer_config.go -f resources.go -f secret.go -f service.go -f storage.go -f split_horizon_dns_config.go -f olm_subscription.go -f pod.go -s Kubernetes -i KubernetesConnector -p kubernetes -o kubernetes_interface.gen.go
