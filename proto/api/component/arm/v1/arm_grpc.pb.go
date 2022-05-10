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

// ArmServiceClient is the client API for ArmService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ArmServiceClient interface {
	// GetEndPosition gets the current position the end of the robot's arm expressed as X,Y,Z,ox,oy,oz,theta
	GetEndPosition(ctx context.Context, in *GetEndPositionRequest, opts ...grpc.CallOption) (*GetEndPositionResponse, error)
	// MoveToPosition moves the mount point of the robot's end effector to the requested position.
	// This will block until done or a new operation cancels this one
	MoveToPosition(ctx context.Context, in *MoveToPositionRequest, opts ...grpc.CallOption) (*MoveToPositionResponse, error)
	// GetJointPositions lists the joint positions (in degrees) of every joint on a robot
	GetJointPositions(ctx context.Context, in *GetJointPositionsRequest, opts ...grpc.CallOption) (*GetJointPositionsResponse, error)
	// MoveToJointPositions moves every joint on a robot's arm to specified angles which are expressed in degrees
	// This will block until done or a new operation cancels this one
	MoveToJointPositions(ctx context.Context, in *MoveToJointPositionsRequest, opts ...grpc.CallOption) (*MoveToJointPositionsResponse, error)
}

type armServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewArmServiceClient(cc grpc.ClientConnInterface) ArmServiceClient {
	return &armServiceClient{cc}
}

func (c *armServiceClient) GetEndPosition(ctx context.Context, in *GetEndPositionRequest, opts ...grpc.CallOption) (*GetEndPositionResponse, error) {
	out := new(GetEndPositionResponse)
	err := c.cc.Invoke(ctx, "/proto.api.component.arm.v1.ArmService/GetEndPosition", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *armServiceClient) MoveToPosition(ctx context.Context, in *MoveToPositionRequest, opts ...grpc.CallOption) (*MoveToPositionResponse, error) {
	out := new(MoveToPositionResponse)
	err := c.cc.Invoke(ctx, "/proto.api.component.arm.v1.ArmService/MoveToPosition", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *armServiceClient) GetJointPositions(ctx context.Context, in *GetJointPositionsRequest, opts ...grpc.CallOption) (*GetJointPositionsResponse, error) {
	out := new(GetJointPositionsResponse)
	err := c.cc.Invoke(ctx, "/proto.api.component.arm.v1.ArmService/GetJointPositions", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *armServiceClient) MoveToJointPositions(ctx context.Context, in *MoveToJointPositionsRequest, opts ...grpc.CallOption) (*MoveToJointPositionsResponse, error) {
	out := new(MoveToJointPositionsResponse)
	err := c.cc.Invoke(ctx, "/proto.api.component.arm.v1.ArmService/MoveToJointPositions", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ArmServiceServer is the server API for ArmService service.
// All implementations must embed UnimplementedArmServiceServer
// for forward compatibility
type ArmServiceServer interface {
	// GetEndPosition gets the current position the end of the robot's arm expressed as X,Y,Z,ox,oy,oz,theta
	GetEndPosition(context.Context, *GetEndPositionRequest) (*GetEndPositionResponse, error)
	// MoveToPosition moves the mount point of the robot's end effector to the requested position.
	// This will block until done or a new operation cancels this one
	MoveToPosition(context.Context, *MoveToPositionRequest) (*MoveToPositionResponse, error)
	// GetJointPositions lists the joint positions (in degrees) of every joint on a robot
	GetJointPositions(context.Context, *GetJointPositionsRequest) (*GetJointPositionsResponse, error)
	// MoveToJointPositions moves every joint on a robot's arm to specified angles which are expressed in degrees
	// This will block until done or a new operation cancels this one
	MoveToJointPositions(context.Context, *MoveToJointPositionsRequest) (*MoveToJointPositionsResponse, error)
	mustEmbedUnimplementedArmServiceServer()
}

// UnimplementedArmServiceServer must be embedded to have forward compatible implementations.
type UnimplementedArmServiceServer struct {
}

func (UnimplementedArmServiceServer) GetEndPosition(context.Context, *GetEndPositionRequest) (*GetEndPositionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetEndPosition not implemented")
}
func (UnimplementedArmServiceServer) MoveToPosition(context.Context, *MoveToPositionRequest) (*MoveToPositionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MoveToPosition not implemented")
}
func (UnimplementedArmServiceServer) GetJointPositions(context.Context, *GetJointPositionsRequest) (*GetJointPositionsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetJointPositions not implemented")
}
func (UnimplementedArmServiceServer) MoveToJointPositions(context.Context, *MoveToJointPositionsRequest) (*MoveToJointPositionsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MoveToJointPositions not implemented")
}
func (UnimplementedArmServiceServer) mustEmbedUnimplementedArmServiceServer() {}

// UnsafeArmServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ArmServiceServer will
// result in compilation errors.
type UnsafeArmServiceServer interface {
	mustEmbedUnimplementedArmServiceServer()
}

func RegisterArmServiceServer(s grpc.ServiceRegistrar, srv ArmServiceServer) {
	s.RegisterService(&ArmService_ServiceDesc, srv)
}

func _ArmService_GetEndPosition_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetEndPositionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArmServiceServer).GetEndPosition(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.api.component.arm.v1.ArmService/GetEndPosition",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArmServiceServer).GetEndPosition(ctx, req.(*GetEndPositionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ArmService_MoveToPosition_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MoveToPositionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArmServiceServer).MoveToPosition(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.api.component.arm.v1.ArmService/MoveToPosition",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArmServiceServer).MoveToPosition(ctx, req.(*MoveToPositionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ArmService_GetJointPositions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetJointPositionsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArmServiceServer).GetJointPositions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.api.component.arm.v1.ArmService/GetJointPositions",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArmServiceServer).GetJointPositions(ctx, req.(*GetJointPositionsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ArmService_MoveToJointPositions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MoveToJointPositionsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArmServiceServer).MoveToJointPositions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.api.component.arm.v1.ArmService/MoveToJointPositions",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArmServiceServer).MoveToJointPositions(ctx, req.(*MoveToJointPositionsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ArmService_ServiceDesc is the grpc.ServiceDesc for ArmService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ArmService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.api.component.arm.v1.ArmService",
	HandlerType: (*ArmServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetEndPosition",
			Handler:    _ArmService_GetEndPosition_Handler,
		},
		{
			MethodName: "MoveToPosition",
			Handler:    _ArmService_MoveToPosition_Handler,
		},
		{
			MethodName: "GetJointPositions",
			Handler:    _ArmService_GetJointPositions_Handler,
		},
		{
			MethodName: "MoveToJointPositions",
			Handler:    _ArmService_MoveToJointPositions_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/api/component/arm/v1/arm.proto",
}
