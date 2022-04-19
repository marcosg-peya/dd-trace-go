// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package pprofutils

import (
	"errors"
	"github.com/google/pprof/profile"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
	"runtime/debug"
	"unsafe"
)

// Delta describes how to compute the delta between two profiles and implements
// the conversion.
type Delta struct {
	// SampleTypes limits the delta calcultion to the given sample types. Other
	// sample types will retain the values of profile b. The defined sample types
	// must exist in the profile, otherwise derivation will fail with an error.
	// If the slice is empty, all sample types are subject to delta profile
	// derivation.
	//
	// The use case for this for this is to deal with the heap profile which
	// contains alloc and inuse sample types, but delta profiling makes no sense
	// for the latter.
	SampleTypes []ValueType
}

// Convert computes the delta between all values b-a and returns them as a new
// profile. Samples that end up with a delta of 0 are dropped. WARNING: Profile
// a will be mutated by this function. You should pass a copy if that's
// undesirable.
func (d Delta) Convert(a, b *profile.Profile) (p *profile.Profile, e error) {
	defer debug.SetPanicOnFault(debug.SetPanicOnFault(true))
	ratios := make([]float64, len(a.SampleType))

	found := 0
	for i, st := range a.SampleType {
		// Empty c.SampleTypes means we calculate the delta for every st
		if len(d.SampleTypes) == 0 {
			ratios[i] = -1
			continue
		}

		// Otherwise we only calcuate the delta for any st that is listed in
		// c.SampleTypes. st's not listed in there will default to ratio 0, which
		// means we delete them from pa, so only the pb values remain in the final
		// profile.
		for _, deltaSt := range d.SampleTypes {
			if deltaSt.Type == st.Type && deltaSt.Unit == st.Unit {
				ratios[i] = -1
				found++
			}
		}
	}
	if found != len(d.SampleTypes) {
		return nil, errors.New("one or more sample type(s) was not found in the profile")
	}

	a.ScaleN(ratios)
	defer func() {
		if r := recover(); r != nil {
			type addressable interface {
				Addr() uintptr
				Error() string
			}
			if a, ok := r.(addressable); ok {
				log.Error("faulty pointer: %v\n", unsafe.Pointer(a.Addr()))
			}
			log.Error("faulty pointer a: %#v, b:%#v\n", a, b)
			for i, s := range a.Sample {
				for j, l := range s.Location {
					log.Error("faulty pointer profile:a sample:%d location:%d locationadd:%v locationval:%#v\n", i, j, unsafe.Pointer(l), *l)
				}
			}
			for i, s := range b.Sample {
				for j, l := range s.Location {
					log.Error("faulty pointer profile:b sample:%d location:%d locationadd:%v locationval:%#v\n", i, j, unsafe.Pointer(l), *l)
				}
			}
			e = r.(error)
		}
	}()
	delta, err := profile.Merge([]*profile.Profile{a, b})
	if err != nil {
		return nil, err
	}
	return delta, delta.CheckValid()
}
