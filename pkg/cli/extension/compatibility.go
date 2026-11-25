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

package extension

import (
	"context"
	"errors"
	"fmt"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	"github.com/openeverest/openeverest/v2/contract"
	"github.com/openeverest/openeverest/v2/pkg/cli"
)

// pluginsCRD is the CRD on which OpenEverest publishes the versions it
// supports for Plugins. Its annotations describe the schema the Plugin is
// applied against, so they are what this cluster actually serves.
const pluginsCRD = "plugins.extensions.openeverest.io"

// compatibility is what installing a Plugin takes, given how it fits the
// OpenEverest in the cluster.
type compatibility struct {
	// notice, when set, is shown before the Plugin is installed.
	notice string
	// acknowledge, when set, is the OpenEverest release to record on the
	// Plugin as the release its incompatibility was acknowledged on.
	acknowledge string
}

// checkCompatibility reads the window of the OpenEverest in the cluster and
// judges plugin against it, before anything is applied. Only a Plugin with
// a valid stamp needs the window, so an unstamped Plugin installs even
// where the CRD cannot be read.
func (pi *PluginInstaller) checkCompatibility(ctx context.Context, plugin *extensionsv1alpha1.Plugin) (compatibility, error) {
	var crd *apiextv1.CustomResourceDefinition
	if contract.Valid(plugin.GetAnnotations()[contract.VersionAnnotation]) {
		c, err := pi.kubeClient.GetCRD(ctx, pluginsCRD)
		switch {
		case apierrors.IsNotFound(err):
			return compatibility{}, fmt.Errorf("cannot install extension %q: OpenEverest is not installed in this cluster (the %s CRD is missing)", plugin.Name, pluginsCRD)
		case err != nil:
			return compatibility{}, fmt.Errorf("cannot read the versions OpenEverest supports from the %s CRD: %w", pluginsCRD, err)
		}
		crd = c
	}
	return judgeCompatibility(plugin, crd, pi.cfg.AcceptIncompatible)
}

// judgeCompatibility decides whether plugin may be installed on the
// OpenEverest whose Plugin CRD is crd. crd is only read when the Plugin's
// stamp is valid; it may be nil otherwise.
//
// The stamp is the OpenEverest version the Plugin was built against, which
// its tooling records on it, and the Plugin is supported when it lies in
// the window the CRD publishes. Outside the window it is refused, unless
// an acknowledgement holds, by contract.AcknowledgementHolds, on the Plugin
// or in the list core publishes, or accept asks to record one, naming the
// release the cluster runs, as contract.AcknowledgedAnnotation: the same
// annotation every other reader of the window honours. A Plugin built for
// an older OpenEverest also installs where core keeps incompatible plugins
// installed (contract.IncompatiblePluginsContinue), since core then keeps
// it switched off rather than refusing it.
//
// An OpenEverest whose CRD publishes no window predates compatibility
// checks. It is refused rather than trusted: it cannot interpret what a
// Plugin built for a newer OpenEverest writes, and nothing can be
// acknowledged on it. A malformed stamp is refused too, as it can never
// become valid. A Plugin without a stamp installs with a notice, because
// plugins are not stamped yet and refusing them would refuse every one.
func judgeCompatibility(plugin *extensionsv1alpha1.Plugin, crd *apiextv1.CustomResourceDefinition, accept bool) (compatibility, error) {
	subject := describe(plugin)
	stamp := plugin.GetAnnotations()[contract.VersionAnnotation]
	if !contract.Valid(stamp) {
		result := contract.Check(stamp, "", "")
		msg := subject + " " + contract.Message(result, stamp, contract.Window{})
		if result.Reason == contract.ReasonUnversioned {
			return compatibility{notice: msg + "; installing it anyway"}, nil
		}
		return compatibility{}, errors.New(msg)
	}

	window, ok := contract.WindowFromAnnotations(crd.GetAnnotations())
	if !ok {
		return compatibility{}, fmt.Errorf("%s cannot be checked for compatibility: the OpenEverest in this "+
			"cluster predates compatibility checks and does not publish the versions it supports; "+
			"upgrade OpenEverest, then install the extension again", subject,
		)
	}
	result := window.Check(stamp)
	if result.Status == metav1.ConditionTrue {
		return compatibility{}, nil
	}

	coreVersion := crd.GetLabels()[contract.CoreVersionKey]
	holds := func(ack string) bool {
		return contract.AcknowledgementHolds(ack, window.Current, coreVersion)
	}
	refusal := refusalMessage(subject, result, stamp, window, coreVersion)

	if acknowledged(plugin, crd, holds) {
		return compatibility{notice: acknowledgedMessage(subject, result, stamp, window) +
			"; the acknowledgement lasts until an OpenEverest upgrade changes compatibility"}, nil
	}
	if accept {
		if !holds(coreVersion) {
			refusal += fmt.Sprintf("; --%s cannot acknowledge it", cli.FlagExtensionAcceptIncompatible)
			if why := whyNotAcknowledgeable(coreVersion, window); why != "" {
				refusal += ", because " + why
			}
			return compatibility{}, errors.New(refusal)
		}
		return compatibility{
			notice: fmt.Sprintf("%s; recording the acknowledgement on the Plugin as %s=%s, where it lasts until an OpenEverest upgrade changes compatibility",
				acknowledgedMessage(subject, result, stamp, window), contract.AcknowledgedAnnotation, coreVersion,
			),
			acknowledge: coreVersion,
		}, nil
	}
	if result.Reason == contract.ReasonExtensionTooOld &&
		crd.GetAnnotations()[contract.IncompatiblePluginsAnnotation] == contract.IncompatiblePluginsContinue {
		return compatibility{notice: refusal +
			"; installing it anyway, because this OpenEverest keeps incompatible plugins installed but does not load them"}, nil
	}

	if holds(coreVersion) {
		refusal += fmt.Sprintf("; pass --%s to install it anyway", cli.FlagExtensionAcceptIncompatible)
	}
	return compatibility{}, errors.New(refusal)
}

// refusalMessage describes to a user why the Plugin subject, stamped with
// stamp, cannot run on an OpenEverest release coreVersion serving window,
// result being the outcome of the check.
func refusalMessage(subject string, result contract.Result, stamp string, window contract.Window, coreVersion string) string {
	msg := subject + " " + contract.Message(result, stamp, window)
	switch {
	case result.Reason == contract.ReasonCoreTooOld && !contract.IsDevelopment(stamp) && contract.Valid(coreVersion):
		msg += " (this cluster runs " + openEverest(coreVersion) + ")"
	case result.Reason == contract.ReasonCoreVersionSkew:
		msg += "; try again once the upgrade has finished"
	}
	return msg
}

// acknowledged reports whether an acknowledgement of plugin holds: the one
// on the Plugin itself, or an entry for it in the list core publishes on
// crd. A list that does not parse acknowledges nothing, as
// contract.ParseAcknowledgedExtensions requires.
func acknowledged(plugin *extensionsv1alpha1.Plugin, crd *apiextv1.CustomResourceDefinition, holds func(string) bool) bool {
	if holds(plugin.GetAnnotations()[contract.AcknowledgedAnnotation]) {
		return true
	}
	acks, err := contract.ParseAcknowledgedExtensions(crd.GetAnnotations()[contract.AcknowledgedExtensionsAnnotation])
	if err != nil {
		return false
	}
	ack, ok := acks[plugin.Name]
	return ok && holds(ack)
}

// acknowledgedMessage describes a Plugin installed only because its
// incompatibility is acknowledged. contract.Acknowledge marks the two
// unsupported reasons; a result that cannot be compared says so itself.
func acknowledgedMessage(subject string, result contract.Result, stamp string, window contract.Window) string {
	acked := contract.Acknowledge(result)
	msg := subject + " " + contract.Message(acked, stamp, window)
	if acked.Reason == result.Reason {
		msg += "; allowed anyway because the incompatibility was acknowledged"
	}
	return msg
}

// whyNotAcknowledgeable explains why no acknowledgement can hold on the
// OpenEverest release coreVersion, which serves window. An acknowledgement
// names that release, and holds only while the release is at least the
// version the window ends at, which an unreleased build is not. It is
// empty when the window is invalid, which the refusal already says.
func whyNotAcknowledgeable(coreVersion string, window contract.Window) string {
	switch {
	case !contract.Valid(coreVersion):
		return "the OpenEverest in this cluster does not publish which release it is"
	case !contract.Valid(window.Current):
		return ""
	default:
		return "this cluster runs an unreleased OpenEverest build"
	}
}

// openEverest names the OpenEverest release v to a user. A development
// value names no release anybody can install.
func openEverest(v string) string {
	if contract.IsDevelopment(v) {
		return "an unreleased OpenEverest build"
	}
	return "OpenEverest " + v
}

// describe names plugin, with its version when it declares one, as the
// subject of a message.
func describe(plugin *extensionsv1alpha1.Plugin) string {
	if plugin.Spec.Version != "" {
		return fmt.Sprintf("extension %q %s", plugin.Name, plugin.Spec.Version)
	}
	return fmt.Sprintf("extension %q", plugin.Name)
}
