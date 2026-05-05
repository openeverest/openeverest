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

// Package cli contains the logic for the CLI commands.
package cli

const (
	// global flags

	// FlagKubeconfig is the name of the kubeconfig flag.
	FlagKubeconfig = "kubeconfig"
	// FlagVerbose is the name of the verbose flag.
	FlagVerbose = "verbose"
	// FlagJSON is the name of the json flag.
	FlagJSON = "json"

	// `install` flags

	// FlagOperatorPostgresql represents the pg operator flag.
	FlagOperatorPostgresql = "operator.postgresql"
	// FlagOperatorXtraDBCluster represents the pxc operator flag.
	FlagOperatorXtraDBCluster = "operator.xtradb-cluster"
	// FlagOperatorMySQL represents the MySQL operator flag.
	FlagOperatorMySQL = "operator.mysql"
	// FlagOperatorMongoDB represents the psmdb operator flag.
	FlagOperatorMongoDB = "operator.mongodb"
	// FlagNamespaces represents the namespaces flag.
	FlagNamespaces = "namespaces"
	// FlagVersionMetadataURL represents the version service url flag.
	FlagVersionMetadataURL = "version-metadata-url"
	// FlagVersion represents the version flag.
	FlagVersion = "version"
	// FlagSkipWizard represents the flag to skip the installation wizard.
	FlagSkipWizard = "skip-wizard"
	// FlagSkipEnvDetection is the name of the skip env detection flag.
	FlagSkipEnvDetection = "skip-env-detection"
	// FlagDisableTelemetry disables telemetry.
	FlagDisableTelemetry = "disable-telemetry"
	// FlagInstallSkipDBNamespace is the name of the skip-db-namespace flag.
	FlagInstallSkipDBNamespace = "skip-db-namespace"

	// `namespaces` flags

	// FlagTakeNamespaceOwnership is the name of the take-ownership flag.
	FlagTakeNamespaceOwnership = "take-ownership"
	// FlagKeepNamespace is the name of the keep-namespace flag.
	FlagKeepNamespace = "keep-namespace"
	// FlagNamespaceForce is the name of the force flag.
	FlagNamespaceForce = "force"
	// FlagNamespaceAll is the name of the all flag.
	FlagNamespaceAll = "all"

	// `upgrade` flags

	// FlagUpgradeDryRun is the name of the dry-run flag.
	FlagUpgradeDryRun = "dry-run"
	// FlagUpgradeInCluster is the name of the in-cluster flag.
	FlagUpgradeInCluster = "in-cluster"
	// FlagUpgradeVersionToUpgrade is the name of the version flag.
	FlagUpgradeVersionToUpgrade = "version"

	// `accounts` flags

	// FlagAccountsUsername is the name of the username flag.
	FlagAccountsUsername = "username"
	// FlagAccountsCreatePassword is the name of the password flag.
	FlagAccountsCreatePassword = "password"
	// FlagAccountsNewPassword is the name of the new-password flag.
	FlagAccountsNewPassword = "new-password"

	// settings flags

	// FlagOIDCIssuerURL is the name of the issuer-url flag.
	FlagOIDCIssuerURL = "issuer-url"
	// FlagOIDCClientID is the name of the client-id flag.
	FlagOIDCClientID = "client-id"
	// FlagOIDCScopes is the name of the scope flag.
	FlagOIDCScopes = "scopes"
	// FlagRBACPolicyFile is the name of the policy-file flag.
	FlagRBACPolicyFile = "policy-file"
)
