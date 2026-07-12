package runtime

import (
	"fmt"

	"google.golang.org/grpc"

	identityinterfaces "logic-grpc-service/internal/identity/interfaces"
	"logic-grpc-service/recruitment/pb"
)

type Deps struct {
	Auth  identityinterfaces.AuthAPI
	Admin identityinterfaces.AdminAPI
	Audit identityinterfaces.AuditAPI
}

type Runtime struct {
	Descriptor Descriptor
	Auth       pb.AuthServiceServer
	Admin      pb.AdminServiceServer
}

func New(deps Deps) (*Runtime, error) {
	descriptor, err := NewDescriptor()
	if err != nil {
		return nil, err
	}
	authServer, err := identityinterfaces.NewAuthServer(deps.Auth)
	if err != nil {
		return nil, err
	}
	adminServer, err := identityinterfaces.NewAdminServer(deps.Admin, deps.Audit)
	if err != nil {
		return nil, err
	}
	runtime := &Runtime{
		Descriptor: descriptor,
		Auth:       authServer,
		Admin:      adminServer,
	}
	if err := runtime.Validate(); err != nil {
		return nil, err
	}
	return runtime, nil
}

func (r *Runtime) Validate() error {
	if r == nil {
		return fmt.Errorf("identity runtime is required")
	}
	if err := Validate(r.Descriptor); err != nil {
		return err
	}
	if r.Auth == nil {
		return fmt.Errorf("identity auth server is required")
	}
	if r.Admin == nil {
		return fmt.Errorf("identity admin server is required")
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
	pb.RegisterAuthServiceServer(registrar, r.Auth)
	pb.RegisterAdminServiceServer(registrar, r.Admin)
	return nil
}
