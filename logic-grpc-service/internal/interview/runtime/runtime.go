package runtime

import (
	"fmt"

	"google.golang.org/grpc"

	interviewinterfaces "logic-grpc-service/internal/interview/interfaces"
	"logic-grpc-service/recruitment/pb"
)

type Deps struct {
	Interview interviewinterfaces.InterviewAPI
}

type Runtime struct {
	Descriptor Descriptor
	Interview  pb.InterviewServiceServer
}

func New(deps Deps) (*Runtime, error) {
	descriptor, err := NewDescriptor()
	if err != nil {
		return nil, err
	}
	interviewServer, err := interviewinterfaces.NewInterviewServer(deps.Interview)
	if err != nil {
		return nil, err
	}
	runtime := &Runtime{
		Descriptor: descriptor,
		Interview:  interviewServer,
	}
	if err := runtime.Validate(); err != nil {
		return nil, err
	}
	return runtime, nil
}

func (r *Runtime) Validate() error {
	if r == nil {
		return fmt.Errorf("interview runtime is required")
	}
	if err := Validate(r.Descriptor); err != nil {
		return err
	}
	if r.Interview == nil {
		return fmt.Errorf("interview server is required")
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
	pb.RegisterInterviewServiceServer(registrar, r.Interview)
	return nil
}
