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

// Package versionservice provides an interface for the Perocona version service.
package versionservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	perconavs "github.com/Percona-Lab/percona-version-service/versionpb"
	goversion "github.com/hashicorp/go-version"
	"google.golang.org/protobuf/encoding/protojson"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

const (
	// PXCOperatorName is the name of the PXC operator in the version service.
	PXCOperatorName = "pxc-operator"
	// PSMDBOperatorName is the name of the PSMDB operator in the version service.
	PSMDBOperatorName = "psmdb-operator"
	// PGOperatorName is the name of the PG operator in the version service.
	PGOperatorName = "pg-operator"
)

// EngineTypeToOperatorName maps an engine type to the operator name in the version service.
//
//nolint:gochecknoglobals
var EngineTypeToOperatorName = map[everestv1alpha1.EngineType]string{
	everestv1alpha1.DatabaseEnginePXC:        PXCOperatorName,
	everestv1alpha1.DatabaseEnginePSMDB:      PSMDBOperatorName,
	everestv1alpha1.DatabaseEnginePostgresql: PGOperatorName,
}

// Interface is the interface for the version service client.
type Interface interface {
	GetSupportedEngineVersions(ctx context.Context, operator, version string) ([]string, error)
	GetEverestMetadata(ctx context.Context) (*perconavs.MetadataResponse, error)
}

type versionServiceClient struct {
	url string
}

// New returns a new version service client.
func New(url string) Interface { //nolint:ireturn
	return &versionServiceClient{url: url}
}

//nolint:gochecknoglobals
var defaultUnmarshal = protojson.UnmarshalOptions{
	// We must ignore any unknown fields in the response, since new fields
	// may be added after this binary has been built and compiled.
	// Without this field, unmashalling would always fail if the API is updated.
	DiscardUnknown: true,
}

// GetSupportedEngineVersions returns a list of supported versions for a given operator and version.
func (c *versionServiceClient) GetSupportedEngineVersions(ctx context.Context, operator, version string) ([]string, error) {
	p, err := url.Parse(c.url)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not parse version service URL"))
	}
	version = strings.TrimPrefix(version, "v")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.JoinPath("versions/v1", operator, version).String(), nil)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not create version service request"))
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not retrieve version response"))
	}
	defer res.Body.Close() //nolint:errcheck

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid response from version service endpoint http %d", res.StatusCode)
	}
	response := &perconavs.VersionResponse{}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not read version response"))
	}
	if err := defaultUnmarshal.Unmarshal(b, response); err != nil {
		return nil, errors.Join(err, errors.New("could not decode version response"))
	}

	if len(response.GetVersions()) == 0 {
		return nil, errors.New("no versions found")
	}

	var versions map[string]*perconavs.Version
	switch operator {
	case PXCOperatorName:
		versions = response.GetVersions()[0].GetMatrix().GetPxc()
	case PSMDBOperatorName:
		versions = response.GetVersions()[0].GetMatrix().GetMongod()
	case PGOperatorName:
		versions = response.GetVersions()[0].GetMatrix().GetPostgresql()
	}

	result := make([]string, 0, len(versions))
	for ver := range versions {
		if _, err := goversion.NewVersion(ver); err != nil {
			return nil, err
		}
		result = append(result, ver)
	}

	// For PXC, we need to remove the 5.x versions from the list
	if operator == PXCOperatorName {
		result = slices.DeleteFunc(result, func(v string) bool {
			semver, _ := goversion.NewVersion(v)
			return semver.Segments()[0] == 5 //nolint:mnd
		})
	}
	slices.Sort(result)
	return result, nil
}

// GetEverestMetadata returns the Everest metadata from the version service.
func (c *versionServiceClient) GetEverestMetadata(ctx context.Context) (*perconavs.MetadataResponse, error) {
	p, err := url.Parse(c.url)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not parse version Everest metadata URL"))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.JoinPath("metadata/v1/everest").String(), nil)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not create Everest metadata request"))
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not retrieve Everest metadata"))
	}
	defer res.Body.Close() //nolint:errcheck

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid response from Everest metadata endpoint http %d", res.StatusCode)
	}
	requirements := &perconavs.MetadataResponse{}
	if err = json.NewDecoder(res.Body).Decode(requirements); err != nil {
		return nil, errors.Join(err, errors.New("could not decode requirements from Everest metadata"))
	}
	return requirements, nil
}
