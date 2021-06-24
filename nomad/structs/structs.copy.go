package structs



func (j *Job) Copy() *Job {

	if j == nil {
		return nil
	}
	xx := new(Job)
	*xx = *j
      
        xx.Datacenters = helper.CopySliceString(j.Datacenters)
      
      
        xx.Constraints = make([]*Constraint, len(j))
        for _, v := range jConstraints {
            xx.Constraints = append(xx.Constraints, v.Copy())
        }
      
      
        xx.Affinities = make([]*Affinity, len(j))
        for _, v := range jAffinities {
            xx.Affinities = append(xx.Affinities, v.Copy())
        }
      
      
        xx.Spreads = make([]*Spread, len(j))
        for _, v := range jSpreads {
            xx.Spreads = append(xx.Spreads, v.Copy())
        }
      
      
        xx.TaskGroups = make([]*TaskGroup, len(j))
        for _, v := range jTaskGroups {
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
