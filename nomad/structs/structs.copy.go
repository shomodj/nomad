package structs

import (
	"github.com/hashicorp/nomad/helper"
)

func (j *Job) Copy() *Job {

	if j == nil {
		return nil
	}
	xx := new(Job)
	*xx = *j

	xx.Datacenters = helper.CopySliceString(j.Datacenters)

	xx.Constraints = make([]*Constraint, len(j.Constraints))
	for _, v := range j.Constraints {
		xx.Constraints = append(xx.Constraints, v)
	}

	xx.Affinities = make([]*Affinity, len(j.Affinities))
	for _, v := range j.Affinities {
		xx.Affinities = append(xx.Affinities, v)
	}

	xx.Spreads = make([]*Spread, len(j.Spreads))
	for _, v := range j.Spreads {
		xx.Spreads = append(xx.Spreads, v)
	}

	xx.TaskGroups = make([]*TaskGroup, len(j.TaskGroups))
	for _, v := range j.TaskGroups {
		xx.TaskGroups = append(xx.TaskGroups, v)
	}

	if j.Multiregion == nil {
		xx.Multiregion = nil
	} else {
		xx.Multiregion = new(pointer)
		*xx.Multiregion = *j.Multiregion
	}

	if j.Periodic == nil {
		xx.Periodic = nil
	} else {
		xx.Periodic = new(pointer)
		*xx.Periodic = *j.Periodic
	}

	if j.ParameterizedJob == nil {
		xx.ParameterizedJob = nil
	} else {
		xx.ParameterizedJob = new(pointer)
		*xx.ParameterizedJob = *j.ParameterizedJob
	}

	xx.Payload = make([]byte, len(j.Payload))
	for _, v := range j.Payload {
		xx.Payload = append(xx.Payload, v)
	}

	xx.Meta = helper.CopyMapStringString(j.Meta)

	return xx
}
