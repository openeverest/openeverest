// everest
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

package preset

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

// UpdateOptions holds the inputs for the update command. The API replaces the
// entire resource (PUT), so -f is a full InstancePreset manifest.
type UpdateOptions struct {
	Name      string // optional override for metadata.name
	Namespace string // optional override for metadata.namespace
	Cluster   string
	Context   string
	File      string // required path to an InstancePreset YAML/JSON manifest
}

// Updater implements `preset update` business logic.
type Updater struct {
	config Config
	l      *zap.SugaredLogger
}

// NewUpdater returns a new Updater.
func NewUpdater(cfg Config, l *zap.SugaredLogger) *Updater {
	u := &Updater{config: cfg, l: l.With("component", "preset-updater")}
	if cfg.Pretty {
		u.l = zap.NewNop().Sugar()
	}
	return u
}

// Run replaces an InstancePreset with the manifest from -f. The live
// resourceVersion is stamped onto the body so a git-exported file without one
// (or with a stale one) still updates cleanly.
func (u *Updater) Run(ctx context.Context, opts UpdateOptions, cfgPath string) error {
	if opts.File == "" {
		return fmt.Errorf("--%s is required", cli.FlagPresetFile)
	}

	preset, err := loadPresetFile(opts.File)
	if err != nil {
		return err
	}
	name, namespace, err := applyIdentity(preset, opts.Name, opts.Namespace)
	if err != nil {
		return err
	}

	c, err := authcli.NewAPIClient(authcli.Config{Pretty: u.config.Pretty}, u.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	live, err := u.fetch(ctx, c, opts.Cluster, namespace, name)
	if err != nil {
		return err
	}

	stripServerFields(preset)
	if live.Metadata != nil {
		preset.Metadata.ResourceVersion = live.Metadata.ResourceVersion
		preset.Metadata.UID = live.Metadata.UID
	}

	resp, err := c.UpdateInstancePresetWithResponse(ctx, opts.Cluster, namespace, name, *preset)
	if err != nil {
		return fmt.Errorf("update instance preset request failed: %w", err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		u.l.Infof("updated instance preset %q in namespace %q", name, namespace)
		return u.emitUpdated(resp.JSON200, name)
	case http.StatusNotFound:
		return fmt.Errorf("instance preset %q not found in namespace %q", name, namespace)
	default:
		if msg, ok := clienterr.Message(resp.JSON400, resp.JSON403, clienterr.FromAPIError(resp.JSON409), clienterr.FromAPIError(resp.JSON422), resp.JSONDefault); ok {
			return fmt.Errorf("server error: %s", msg)
		}
		return fmt.Errorf("unexpected response updating instance preset %q: %s", name, resp.Status())
	}
}

func (u *Updater) fetch(ctx context.Context, c *client.ClientWithResponses, cluster, namespace, name string) (*client.InstancePreset, error) {
	resp, err := c.GetInstancePresetWithResponse(ctx, cluster, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch instance preset %q: %w", name, err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("instance preset %q not found in namespace %q", name, namespace)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		if msg, ok := clienterr.Message(resp.JSON403, resp.JSONDefault); ok {
			return nil, fmt.Errorf("server error: %s", msg)
		}
		return nil, fmt.Errorf("unexpected response fetching instance preset %q: %s", name, resp.Status())
	}
	return resp.JSON200, nil
}

func (u *Updater) emitUpdated(updated *client.InstancePreset, name string) error {
	if u.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Instance preset %q updated", name))
		return nil
	}
	if updated == nil {
		return fmt.Errorf("instance preset %q was updated but the server returned an unreadable response body", name)
	}
	return cli.Encode(os.Stdout, updated, u.config.YAML)
}
