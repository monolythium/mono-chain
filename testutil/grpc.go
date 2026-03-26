package testutil

import "google.golang.org/grpc"

// MockServiceRegistrar satisfies grpc.ServiceRegistrar for RegisterServices tests.
type MockServiceRegistrar struct{}

func (MockServiceRegistrar) RegisterService(*grpc.ServiceDesc, any) {}
