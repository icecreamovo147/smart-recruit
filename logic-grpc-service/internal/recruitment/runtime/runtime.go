package runtime

import (
	"fmt"

	"google.golang.org/grpc"

	recruitmentinterfaces "logic-grpc-service/internal/recruitment/interfaces"
	"logic-grpc-service/recruitment/pb"
)

type Deps struct {
	Job         recruitmentinterfaces.JobAPI
	JobTaxonomy recruitmentinterfaces.JobTaxonomyAPI
	Candidate   recruitmentinterfaces.CandidateProfileAPI
	Application recruitmentinterfaces.ApplicationAPI
}

type Runtime struct {
	Descriptor  Descriptor
	Job         pb.JobServiceServer
	Candidate   pb.CandidateServiceServer
	Application pb.ApplicationServiceServer
}

func New(deps Deps) (*Runtime, error) {
	descriptor, err := NewDescriptor()
	if err != nil {
		return nil, err
	}
	jobServer, err := recruitmentinterfaces.NewJobServer(deps.Job, deps.JobTaxonomy)
	if err != nil {
		return nil, err
	}
	candidateServer, err := recruitmentinterfaces.NewCandidateServer(deps.Candidate)
	if err != nil {
		return nil, err
	}
	applicationServer, err := recruitmentinterfaces.NewApplicationServer(deps.Application)
	if err != nil {
		return nil, err
	}
	runtime := &Runtime{
		Descriptor:  descriptor,
		Job:         jobServer,
		Candidate:   candidateServer,
		Application: applicationServer,
	}
	if err := runtime.Validate(); err != nil {
		return nil, err
	}
	return runtime, nil
}

func (r *Runtime) Validate() error {
	if r == nil {
		return fmt.Errorf("recruitment runtime is required")
	}
	if err := Validate(r.Descriptor); err != nil {
		return err
	}
	if r.Job == nil {
		return fmt.Errorf("recruitment job server is required")
	}
	if r.Candidate == nil {
		return fmt.Errorf("recruitment candidate server is required")
	}
	if r.Application == nil {
		return fmt.Errorf("recruitment application server is required")
	}
	return nil
}

func (r *Runtime) RegisterGRPC(registrar grpc.ServiceRegistrar) error {
	if registrar == nil {
		return fmt.Errorf("grpc service registrar is required")
	}
	if err := r.Validate(); err != nil {
		return err
	}
	pb.RegisterJobServiceServer(registrar, r.Job)
	pb.RegisterCandidateServiceServer(registrar, r.Candidate)
	pb.RegisterApplicationServiceServer(registrar, r.Application)
	return nil
}
