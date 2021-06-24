package structs

func (j *Job) Equals(other *Job) bool {

	if j.Stop != other.Stop {
		return false
	}
	if j.Region != other.Region {
		return false
	}
	if j.Namespace != other.Namespace {
		return false
	}
	if j.ID != other.ID {
		return false
	}
	if j.ParentID != other.ParentID {
		return false
	}
	if j.Name != other.Name {
		return false
	}
	if j.Type != other.Type {
		return false
	}
	if j.Priority != other.Priority {
		return false
	}
	if j.AllAtOnce != other.AllAtOnce {
		return false
	}
	if j.Update != other.Update {
		return false
	}
	if !j.Multiregion.Equals(&other.Multiregion) {
		return false
	}
	if !j.Periodic.Equals(&other.Periodic) {
		return false
	}
	if !j.ParameterizedJob.Equals(&other.ParameterizedJob) {
		return false
	}
	if j.Dispatched != other.Dispatched {
		return false
	}
	if j.ConsulToken != other.ConsulToken {
		return false
	}
	if j.ConsulNamespace != other.ConsulNamespace {
		return false
	}
	if j.VaultToken != other.VaultToken {
		return false
	}
	if j.VaultNamespace != other.VaultNamespace {
		return false
	}
	if j.NomadTokenID != other.NomadTokenID {
		return false
	}
	if j.Status != other.Status {
		return false
	}
	if j.StatusDescription != other.StatusDescription {
		return false
	}
	if j.Stable != other.Stable {
		return false
	}
	if j.Version != other.Version {
		return false
	}
	if j.SubmitTime != other.SubmitTime {
		return false
	}
	if j.CreateIndex != other.CreateIndex {
		return false
	}
	if j.ModifyIndex != other.ModifyIndex {
		return false
	}
	if j.JobModifyIndex != other.JobModifyIndex {
		return false
	}

	return true
}
