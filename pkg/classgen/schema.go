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

package classgen

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/tools/go/packages"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-tools/pkg/crd"
	crdmarkers "sigs.k8s.io/controller-tools/pkg/crd/markers"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"
)

// ClosedMarker is the name of the marker that closes the object a struct
// type renders to: +openeverest:validation:Closed sets
// additionalProperties: false on it.
const ClosedMarker = "openeverest:validation:Closed"

// Packages are Go packages loaded once to render the schemas of several of
// their types. A Packages is not safe for concurrent use.
type Packages struct {
	patterns []string
	roots    []*loader.Package
	parser   *crd.Parser
}

// Load loads the packages matching pkgPatterns, which are relative paths such
// as ./api or import paths, as the go command resolves them in the working
// directory. A package that does not compile is an error.
func Load(pkgPatterns ...string) (*Packages, error) {
	if len(pkgPatterns) == 0 {
		return nil, errors.New("no package to load")
	}
	roots, err := loader.LoadRoots(pkgPatterns...)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", strings.Join(pkgPatterns, ", "), err)
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("no package matches %s", strings.Join(pkgPatterns, ", "))
	}

	registry := &markers.Registry{}
	if err := crdmarkers.Register(registry); err != nil {
		return nil, fmt.Errorf("registering the kubebuilder markers: %w", err)
	}
	closedDef, err := markers.MakeDefinition(ClosedMarker, markers.DescribesType, closed{})
	if err != nil {
		return nil, err
	}
	if err := registry.Register(closedDef); err != nil {
		return nil, fmt.Errorf("registering +%s: %w", ClosedMarker, err)
	}
	parser := &crd.Parser{
		Collector: &markers.Collector{Registry: registry},
		Checker:   &loader.TypeChecker{},
		// Parameters are JSON payloads rather than stored API objects, so
		// floats are allowed.
		AllowDangerousTypes: true,
	}
	crd.AddKnownTypes(parser)
	checked := make(map[*loader.Package]bool)
	for _, root := range roots {
		for _, imp := range root.Imports() {
			typeCheck(imp, checked)
		}
		parser.NeedPackage(root)
	}
	if err := packageErrors(roots); err != nil {
		return nil, err
	}
	return &Packages{patterns: pkgPatterns, roots: roots, parser: parser}, nil
}

// Schema loads pkgPatterns and returns the schema of the parameters type
// typeName. See [Packages.Schema].
func Schema(pkgPatterns []string, typeName string) (*apiextensionsv1.JSONSchemaProps, error) {
	p, err := Load(pkgPatterns...)
	if err != nil {
		return nil, err
	}
	return p.Schema(typeName)
}

// Schema returns the OpenAPI v3 schema of typeName, a parameters type: the
// schema controller-gen writes for a CRD field of that type, with its markers
// and doc comments, flattened, and without the root description, which
// documents the Go type rather than the parameters. The type must render to
// an object.
//
// x-kubernetes-validations anywhere in it are an error: parameters are
// validated by a JSON schema validator, which does not evaluate CEL. Rules on
// parameters belong on a view (see [Packages.View]).
func (p *Packages) Schema(typeName string) (*apiextensionsv1.JSONSchemaProps, error) {
	s, err := p.schema(typeName)
	if err != nil {
		return nil, err
	}
	if s.Type != "object" {
		return nil, fmt.Errorf("%s renders to type %q: parameters are an object", typeName, s.Type)
	}
	if err := rejectCEL(typeName, s); err != nil {
		return nil, err
	}
	return s, nil
}

// View returns the OpenAPI v3 schema of typeName, a view type: the JSON shape
// of the spec of the objects that reference a class, whose root XValidation
// markers are the class's validations. Pass it to [Validations].
func (p *Packages) View(typeName string) (*apiextensionsv1.JSONSchemaProps, error) {
	return p.schema(typeName)
}

func (p *Packages) schema(typeName string) (*apiextensionsv1.JSONSchemaProps, error) {
	ident, err := p.lookup(typeName)
	if err != nil {
		return nil, err
	}
	p.parser.NeedFlattenedSchemaFor(ident)
	if err := packageErrors(p.roots); err != nil {
		return nil, err
	}
	s := p.parser.FlattenedSchemata[ident]
	out := s.DeepCopy()
	out.Description = ""
	return out, nil
}

// lookup finds the root package that declares typeName. A name declared in
// several of them is ambiguous.
func (p *Packages) lookup(typeName string) (crd.TypeIdent, error) {
	var found []crd.TypeIdent
	for _, root := range p.roots {
		if p.parser.LookupType(root, typeName) != nil {
			found = append(found, crd.TypeIdent{Package: root, Name: typeName})
		}
	}
	switch len(found) {
	case 0:
		return crd.TypeIdent{}, fmt.Errorf("type %s not found in %s", typeName, strings.Join(p.patterns, ", "))
	case 1:
		return found[0], nil
	default:
		paths := make([]string, 0, len(found))
		for _, f := range found {
			paths = append(paths, f.Package.PkgPath)
		}
		return crd.TypeIdent{}, fmt.Errorf("type %s is declared in several packages: %s", typeName, strings.Join(paths, ", "))
	}
}

// typeCheck type-checks pkg after its imports. The type checker of
// controller-tools checks only the packages a root's type declarations
// reference and leaves the root's other imports empty, so a root whose
// functions or variables use them, as a contract's decoders do, would report
// them undefined.
func typeCheck(pkg *loader.Package, checked map[*loader.Package]bool) {
	if checked[pkg] || pkg.PkgPath == "unsafe" {
		return
	}
	checked[pkg] = true
	for _, imp := range pkg.Imports() {
		typeCheck(imp, checked)
	}
	pkg.NeedTypesInfo()
}

// packageErrors returns every error recorded while loading packages and
// rendering schemas: any error in the root packages, and anything but
// type-checking errors in the packages they import, as controller-gen does.
func packageErrors(roots []*loader.Package) error {
	isRoot := make(map[*packages.Package]bool, len(roots))
	raw := make([]*packages.Package, 0, len(roots))
	for _, root := range roots {
		isRoot[root.Package] = true
		raw = append(raw, root.Package)
	}

	var errs []error
	seen := make(map[string]bool)
	packages.Visit(raw, nil, func(pkg *packages.Package) {
		for _, err := range pkg.Errors {
			// The loader records the first type-checking error twice.
			if (err.Kind == packages.TypeError && !isRoot[pkg]) || seen[err.Error()] {
				continue
			}
			seen[err.Error()] = true
			errs = append(errs, err)
		}
	})
	return errors.Join(errs...)
}

// rejectCEL fails on x-kubernetes-validations in s or below it.
func rejectCEL(path string, s *apiextensionsv1.JSONSchemaProps) error {
	if len(s.XValidations) > 0 {
		return fmt.Errorf("%s: x-kubernetes-validations are not evaluated on parameters; put the rule on the root of a view type", path)
	}
	var errs []error
	for name, prop := range s.Properties {
		errs = append(errs, rejectCEL(path+"."+name, &prop))
	}
	if s.Items != nil && s.Items.Schema != nil {
		errs = append(errs, rejectCEL(path+"[]", s.Items.Schema))
	}
	if s.AdditionalProperties != nil && s.AdditionalProperties.Schema != nil {
		errs = append(errs, rejectCEL(path+"{}", s.AdditionalProperties.Schema))
	}
	return errors.Join(errs...)
}

// closed is the +openeverest:validation:Closed marker.
type closed struct{}

// ApplyToSchema implements crd.SchemaMarker.
func (closed) ApplyToSchema(_ *crdmarkers.SchemaContext, s *apiextensionsv1.JSONSchemaProps) error {
	preserves := s.XPreserveUnknownFields != nil && *s.XPreserveUnknownFields
	if s.Type != "object" || s.AdditionalProperties != nil || preserves {
		return fmt.Errorf("+%s applies to struct types", ClosedMarker)
	}
	s.AdditionalProperties = &apiextensionsv1.JSONSchemaPropsOrBool{Allows: false}
	return nil
}
