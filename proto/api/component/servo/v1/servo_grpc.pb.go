// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ServoServiceClient is the client API for ServoService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ServoServiceClient interface {
	// Move requests the servo of the underlying robot to move.
	// This will block until done or a new operation cancels this one
	Move(ctx context.Context, in *MoveRequest, opts ...grpc.CallOption) (*MoveResponse, error)
	// GetPosition returns the current set angle (degrees) of the servo of the underlying robot.
	GetPosition(ctx context.Context, in *GetPositionRequest, opts ...grpc.CallOption) (*GetPositionResponse, error)
}

type servoServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewServoServiceClient(cc grpc.ClientConnInterface) ServoServiceClient {
	return &servoServiceClient{cc}
}

func (c *servoServiceClient) Move(ctx context.Context, in *MoveRequest, opts ...grpc.CallOption) (*MoveResponse, error) {
	out := new(MoveResponse)
	err := c.cc.Invoke(ctx, "/proto.api.component.servo.v1.ServoService/Move", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *servoServiceClient) GetPosition(ctx context.Context, in *GetPositionRequest, opts ...grpc.CallOption) (*GetPositionResponse, error) {
	out := new(GetPositionResponse)
	err := c.cc.Invoke(ctx, "/proto.api.component.servo.v1.ServoService/GetPosition", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ServoServiceServer is the server API for ServoService service.
// All implementations must embed UnimplementedServoServiceServer
// for forward compatibility
type ServoServiceServer interface {
	// Move requests the servo of the underlying robot to move.
	// This will block until done or a new operation cancels this one
	Move(context.Context, *MoveRequest) (*MoveResponse, error)
	// GetPosition returns the current set angle (degrees) of the servo of the underlying robot.
	GetPosition(context.Context, *GetPositionRequest) (*GetPositionResponse, error)
	mustEmbedUnimplementedServoServiceServer()
}

// UnimplementedServoServiceServer must be embedded to have forward compatible implementations.
type UnimplementedServoServiceServer struct {
}

func (UnimplementedServoServiceServer) Move(context.Context, *MoveRequest) (*MoveResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Move not implemented")
}
func (UnimplementedServoServiceServer) GetPosition(context.Context, *GetPositionRequest) (*GetPositionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPosition not implemented")
}
func (UnimplementedServoServiceServer) mustEmbedUnimplementedServoServiceServer() {}

// UnsafeServoServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ServoServiceServer will
// result in compilation errors.
type UnsafeServoServiceServer interface {
	mustEmbedUnimplementedServoServiceServer()
}

func RegisterServoServiceServer(s grpc.ServiceRegistrar, srv ServoServiceServer) {
	s.RegisterService(&ServoService_ServiceDesc, srv)
}

func _ServoService_Move_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MoveRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServoServiceServer).Move(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.api.component.servo.v1.ServoService/Move",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServoServiceServer).Move(ctx, req.(*MoveRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ServoService_GetPosition_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetPositionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServoServiceServer).GetPosition(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.api.component.servo.v1.ServoService/GetPosition",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServoServiceServer).GetPosition(ctx, req.(*GetPositionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ServoService_ServiceDesc is the grpc.ServiceDesc for ServoService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ServoService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.api.component.servo.v1.ServoService",
	HandlerType: (*ServoServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Move",
			Handler:    _ServoService_Move_Handler,
		},
		{
			MethodName: "GetPosition",
			Handler:    _ServoService_GetPosition_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/api/component/servo/v1/servo.proto",
}
