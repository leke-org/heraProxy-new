// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.21.11
// source: protocol/grpc.proto

package protobuf

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

const (
	Auth_SetUserData_FullMethodName    = "/Auth/SetUserData"
	Auth_DeleteUserData_FullMethodName = "/Auth/DeleteUserData"
	Auth_GetUserData_FullMethodName    = "/Auth/GetUserData"
	Auth_AddUserData_FullMethodName    = "/Auth/AddUserData"
	Auth_Disconnect_FullMethodName     = "/Auth/Disconnect"
)

// AuthClient is the client API for Auth service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AuthClient interface {
	SetUserData(ctx context.Context, in *AuthInfo, opts ...grpc.CallOption) (*NullMessage, error)
	DeleteUserData(ctx context.Context, in *AuthInfo, opts ...grpc.CallOption) (*NullMessage, error)
	GetUserData(ctx context.Context, in *AuthInfo, opts ...grpc.CallOption) (*AuthInfo, error)
	AddUserData(ctx context.Context, in *AuthInfo, opts ...grpc.CallOption) (*NullMessage, error)
	Disconnect(ctx context.Context, in *DisconnectInfo, opts ...grpc.CallOption) (*NullMessage, error)
}

type authClient struct {
	cc grpc.ClientConnInterface
}

func NewAuthClient(cc grpc.ClientConnInterface) AuthClient {
	return &authClient{cc}
}

func (c *authClient) SetUserData(ctx context.Context, in *AuthInfo, opts ...grpc.CallOption) (*NullMessage, error) {
	out := new(NullMessage)
	err := c.cc.Invoke(ctx, Auth_SetUserData_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authClient) DeleteUserData(ctx context.Context, in *AuthInfo, opts ...grpc.CallOption) (*NullMessage, error) {
	out := new(NullMessage)
	err := c.cc.Invoke(ctx, Auth_DeleteUserData_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authClient) GetUserData(ctx context.Context, in *AuthInfo, opts ...grpc.CallOption) (*AuthInfo, error) {
	out := new(AuthInfo)
	err := c.cc.Invoke(ctx, Auth_GetUserData_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authClient) AddUserData(ctx context.Context, in *AuthInfo, opts ...grpc.CallOption) (*NullMessage, error) {
	out := new(NullMessage)
	err := c.cc.Invoke(ctx, Auth_AddUserData_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authClient) Disconnect(ctx context.Context, in *DisconnectInfo, opts ...grpc.CallOption) (*NullMessage, error) {
	out := new(NullMessage)
	err := c.cc.Invoke(ctx, Auth_Disconnect_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AuthServer is the server API for Auth service.
// All implementations must embed UnimplementedAuthServer
// for forward compatibility
type AuthServer interface {
	SetUserData(context.Context, *AuthInfo) (*NullMessage, error)
	DeleteUserData(context.Context, *AuthInfo) (*NullMessage, error)
	GetUserData(context.Context, *AuthInfo) (*AuthInfo, error)
	AddUserData(context.Context, *AuthInfo) (*NullMessage, error)
	Disconnect(context.Context, *DisconnectInfo) (*NullMessage, error)
	mustEmbedUnimplementedAuthServer()
}

// UnimplementedAuthServer must be embedded to have forward compatible implementations.
type UnimplementedAuthServer struct {
}

func (UnimplementedAuthServer) SetUserData(context.Context, *AuthInfo) (*NullMessage, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetUserData not implemented")
}
func (UnimplementedAuthServer) DeleteUserData(context.Context, *AuthInfo) (*NullMessage, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteUserData not implemented")
}
func (UnimplementedAuthServer) GetUserData(context.Context, *AuthInfo) (*AuthInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUserData not implemented")
}
func (UnimplementedAuthServer) AddUserData(context.Context, *AuthInfo) (*NullMessage, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddUserData not implemented")
}
func (UnimplementedAuthServer) Disconnect(context.Context, *DisconnectInfo) (*NullMessage, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Disconnect not implemented")
}
func (UnimplementedAuthServer) mustEmbedUnimplementedAuthServer() {}

// UnsafeAuthServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AuthServer will
// result in compilation errors.
type UnsafeAuthServer interface {
	mustEmbedUnimplementedAuthServer()
}

func RegisterAuthServer(s grpc.ServiceRegistrar, srv AuthServer) {
	s.RegisterService(&Auth_ServiceDesc, srv)
}

func _Auth_SetUserData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AuthInfo)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthServer).SetUserData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Auth_SetUserData_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthServer).SetUserData(ctx, req.(*AuthInfo))
	}
	return interceptor(ctx, in, info, handler)
}

func _Auth_DeleteUserData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AuthInfo)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthServer).DeleteUserData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Auth_DeleteUserData_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthServer).DeleteUserData(ctx, req.(*AuthInfo))
	}
	return interceptor(ctx, in, info, handler)
}

func _Auth_GetUserData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AuthInfo)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthServer).GetUserData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Auth_GetUserData_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthServer).GetUserData(ctx, req.(*AuthInfo))
	}
	return interceptor(ctx, in, info, handler)
}

func _Auth_AddUserData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AuthInfo)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthServer).AddUserData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Auth_AddUserData_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthServer).AddUserData(ctx, req.(*AuthInfo))
	}
	return interceptor(ctx, in, info, handler)
}

func _Auth_Disconnect_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DisconnectInfo)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthServer).Disconnect(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Auth_Disconnect_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthServer).Disconnect(ctx, req.(*DisconnectInfo))
	}
	return interceptor(ctx, in, info, handler)
}

// Auth_ServiceDesc is the grpc.ServiceDesc for Auth service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Auth_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "Auth",
	HandlerType: (*AuthServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SetUserData",
			Handler:    _Auth_SetUserData_Handler,
		},
		{
			MethodName: "DeleteUserData",
			Handler:    _Auth_DeleteUserData_Handler,
		},
		{
			MethodName: "GetUserData",
			Handler:    _Auth_GetUserData_Handler,
		},
		{
			MethodName: "AddUserData",
			Handler:    _Auth_AddUserData_Handler,
		},
		{
			MethodName: "Disconnect",
			Handler:    _Auth_Disconnect_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "protocol/grpc.proto",
}

const (
	RemoteAuth_GetUserData_FullMethodName = "/RemoteAuth/GetUserData"
)

// RemoteAuthClient is the client API for RemoteAuth service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RemoteAuthClient interface {
	GetUserData(ctx context.Context, in *AuthInfo, opts ...grpc.CallOption) (*AuthInfo, error)
}

type remoteAuthClient struct {
	cc grpc.ClientConnInterface
}

func NewRemoteAuthClient(cc grpc.ClientConnInterface) RemoteAuthClient {
	return &remoteAuthClient{cc}
}

func (c *remoteAuthClient) GetUserData(ctx context.Context, in *AuthInfo, opts ...grpc.CallOption) (*AuthInfo, error) {
	out := new(AuthInfo)
	err := c.cc.Invoke(ctx, RemoteAuth_GetUserData_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RemoteAuthServer is the server API for RemoteAuth service.
// All implementations must embed UnimplementedRemoteAuthServer
// for forward compatibility
type RemoteAuthServer interface {
	GetUserData(context.Context, *AuthInfo) (*AuthInfo, error)
	mustEmbedUnimplementedRemoteAuthServer()
}

// UnimplementedRemoteAuthServer must be embedded to have forward compatible implementations.
type UnimplementedRemoteAuthServer struct {
}

func (UnimplementedRemoteAuthServer) GetUserData(context.Context, *AuthInfo) (*AuthInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUserData not implemented")
}
func (UnimplementedRemoteAuthServer) mustEmbedUnimplementedRemoteAuthServer() {}

// UnsafeRemoteAuthServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RemoteAuthServer will
// result in compilation errors.
type UnsafeRemoteAuthServer interface {
	mustEmbedUnimplementedRemoteAuthServer()
}

func RegisterRemoteAuthServer(s grpc.ServiceRegistrar, srv RemoteAuthServer) {
	s.RegisterService(&RemoteAuth_ServiceDesc, srv)
}

func _RemoteAuth_GetUserData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AuthInfo)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteAuthServer).GetUserData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RemoteAuth_GetUserData_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteAuthServer).GetUserData(ctx, req.(*AuthInfo))
	}
	return interceptor(ctx, in, info, handler)
}

// RemoteAuth_ServiceDesc is the grpc.ServiceDesc for RemoteAuth service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RemoteAuth_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "RemoteAuth",
	HandlerType: (*RemoteAuthServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetUserData",
			Handler:    _RemoteAuth_GetUserData_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "protocol/grpc.proto",
}
