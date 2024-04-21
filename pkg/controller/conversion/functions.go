// SPDX-FileCopyrightText: 2023 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package conversion

import (
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/crossplane/upjet/pkg/config/conversion"
	"github.com/crossplane/upjet/pkg/resource"
)

// RoundTrip round-trips from `src` to `dst` via an unstructured map[string]any
// representation of the `src` object and applies the registered webhook
// conversion functions of this registry.
func (r *registry) RoundTrip(dst, src resource.Terraformed) error { //nolint:gocyclo // considered breaking this according to the converters and I did not like it
	// first PrioritizedManagedConversions are run in their registration order
	for _, c := range r.GetConversions(dst) {
		if pc, ok := c.(conversion.PrioritizedManagedConversion); ok {
			if _, err := pc.ConvertManaged(src, dst); err != nil {
				return errors.Wrapf(err, "cannot apply the PrioritizedManagedConversion for the %q object", dst.GetTerraformResourceType())
			}
		}
	}

	srcMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(src)
	if err != nil {
		return errors.Wrap(err, "cannot convert the conversion source object into the map[string]any representation")
	}
	// now we will try to run the registered webhook conversions
	dstMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(dst)
	if err != nil {
		return errors.Wrap(err, "cannot convert the conversion destination object into the map[string]any representation")
	}
	srcPaved := fieldpath.Pave(srcMap)
	dstPaved := fieldpath.Pave(dstMap)
	// then run the PavedConversions
	for _, c := range r.GetConversions(dst) {
		if pc, ok := c.(conversion.PavedConversion); ok {
			if _, err := pc.ConvertPaved(srcPaved, dstPaved); err != nil {
				return errors.Wrapf(err, "cannot apply the PavedConversion for the %q object", dst.GetTerraformResourceType())
			}
		}
	}
	// convert the map[string]any representation of the conversion target back to
	// the original type.
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(dstMap, dst); err != nil {
		return errors.Wrap(err, "cannot convert the map[string]any representation of the conversion target back to the object itself")
	}

	// finally at the third stage, run the ManagedConverters
	for _, c := range r.GetConversions(dst) {
		if tc, ok := c.(conversion.ManagedConversion); ok {
			if _, ok := tc.(conversion.PrioritizedManagedConversion); ok {
				continue // then already run in the first stage
			}
			if _, err := tc.ConvertManaged(src, dst); err != nil {
				return errors.Wrapf(err, "cannot apply the ManagedConversion for the %q object", dst.GetTerraformResourceType())
			}
		}
	}

	return nil
}

// RoundTrip round-trips from `src` to `dst` via an unstructured map[string]any
// representation of the `src` object and applies the registered webhook
// conversion functions.
func RoundTrip(dst, src resource.Terraformed) error {
	return instance.RoundTrip(dst, src)
}
