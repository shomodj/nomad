package main
//
//import (
//	"github.com/hashicorp/nomad/nomad/structs"
//)
//
////go:generate nomad-generate -methods all -type Job -exclude Stop -exclude CreateIndex
//
//// Job is the scope of a scheduling request to Nomad. It is the largest
//// scoped object, and is a named collection of task groups. Each task group
//// is further composed of tasks. A task group (TG) is the unit of scheduling
//// however.
//type Job struct {
//	// Stop marks whether the user has stopped the job. A stopped job will
//	// have all created allocations stopped and acts as a way to stop a job
//	// without purging it from the system. This allows existing allocs to be
//	// queried and the job to be inspected as it is being killed.
//	Stop bool
//
//	// Region is the Nomad region that handles scheduling this job
//	Region string
//
//	// Namespace is the namespace the job is submitted into.
//	Namespace string
//
//	// ID is a unique identifier for the job per region. It can be
//	// specified hierarchically like LineOfBiz/OrgName/Team/Project
//	ID string
//
//	// ParentID is the unique identifier of the job that spawned this job.
//	ParentID string
//
//	// Name is the logical name of the job used to refer to it. This is unique
//	// per region, but not unique globally.
//	Name string
//
//	// Type is used to control various behaviors about the job. Most jobs
//	// are service jobs, meaning they are expected to be long lived.
//	// Some jobs are batch oriented meaning they run and then terminate.
//	// This can be extended in the future to support custom schedulers.
//	Type string
//
//	// Priority is used to control scheduling importance and if this job
//	// can preempt other jobs.
//	Priority int
//
//	// AllAtOnce is used to control if incremental scheduling of task groups
//	// is allowed or if we must do a gang scheduling of the entire job. This
//	// can slow down larger jobs if resources are not available.
//	AllAtOnce bool
//
//	// Datacenters contains all the datacenters this job is allowed to span
//	Datacenters []string
//
//	// Constraints can be specified at a job level and apply to
//	// all the task groups and tasks.
//	Constraints []*structs.Constraint
//
//	// Affinities can be specified at the job level to express
//	// scheduling preferences that apply to all groups and tasks
//	Affinities []*structs.Affinity
//
//	// Spread can be specified at the job level to express spreading
//	// allocations across a desired attribute, such as datacenter
//	Spreads []*structs.Spread
//
//	// TaskGroups are the collections of task groups that this job needs
//	// to run. Each task group is an atomic unit of scheduling and placement.
//	TaskGroups []*structs.TaskGroup
//
//	// See agent.ApiJobToStructJob
//	// Update provides defaults for the TaskGroup Update stanzas
//	Update structs.UpdateStrategy
//
//	Multiregion *structs.Multiregion
//
//	// Periodic is used to define the interval the job is run at.
//	Periodic *structs.PeriodicConfig
//
//	// ParameterizedJob is used to specify the job as a parameterized job
//	// for dispatching.
//	ParameterizedJob *structs.ParameterizedJobConfig
//
//	// Dispatched is used to identify if the Job has been dispatched from a
//	// parameterized job.
//	Dispatched bool
//
//	// Payload is the payload supplied when the job was dispatched.
//	Payload []byte
//
//	// Meta is used to associate arbitrary metadata with this
//	// job. This is opaque to Nomad.
//	Meta map[string]string
//
//	// ConsulToken is the Consul token that proves the submitter of the job has
//	// access to the Service Identity policies associated with the job's
//	// Consul Connect enabled services. This field is only used to transfer the
//	// token and is not stored after Job submission.
//	ConsulToken string
//
//	// ConsulNamespace is the Consul namespace
//	ConsulNamespace string
//
//	// VaultToken is the Vault token that proves the submitter of the job has
//	// access to the specified Vault policies. This field is only used to
//	// transfer the token and is not stored after Job submission.
//	VaultToken string
//
//	// VaultNamespace is the Vault namespace
//	VaultNamespace string
//
//	// NomadTokenID is the Accessor ID of the ACL token (if any)
//	// used to register this version of the job. Used by deploymentwatcher.
//	NomadTokenID string
//
//	// Job status
//	Status string
//
//	// StatusDescription is meant to provide more human useful information
//	StatusDescription string
//
//	// Stable marks a job as stable. Stability is only defined on "service" and
//	// "system" jobs. The stability of a job will be set automatically as part
//	// of a deployment and can be manually set via APIs. This field is updated
//	// when the status of a corresponding deployment transitions to Failed
//	// or Successful. This field is not meaningful for jobs that don't have an
//	// update stanza.
//	Stable bool
//
//	// Version is a monotonically increasing version number that is incremented
//	// on each job register.
//	Version uint64
//
//	// SubmitTime is the time at which the job was submitted as a UnixNano in
//	// UTC
//	SubmitTime int64
//
//	// Raft Indexes
//	CreateIndex    uint64
//	ModifyIndex    uint64
//	JobModifyIndex uint64
//}
