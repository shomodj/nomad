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
		xx.Constraints = append(xx.Constraints, v.Copy())
	}

	xx.Affinities = make([]*Affinity, len(j.Affinities))
	for _, v := range j.Affinities {
		xx.Affinities = append(xx.Affinities, v.Copy())
	}

	xx.Spreads = make([]*Spread, len(j.Spreads))
	for _, v := range j.Spreads {
		xx.Spreads = append(xx.Spreads, v.Copy())
	}

	xx.TaskGroups = make([]*TaskGroup, len(j.TaskGroups))
	for _, v := range j.TaskGroups {
		xx.TaskGroups = append(xx.TaskGroups, v.Copy())
	}

	xx.Multiregion = j.Multiregion.Copy()

	xx.Periodic = j.Periodic.Copy()

	xx.ParameterizedJob = j.ParameterizedJob.Copy()

	xx.Payload = make([]byte, len(j.Payload))
	for _, v := range j.Payload {
		xx.Payload = append(xx.Payload, v)
	}

	xx.Meta = helper.CopyMapStringString(j.Meta)

	return xx
}
