// Code generated by protoc-gen-go. DO NOT EDIT.
// source: proto/exsync.proto

/*
Package exsync is a generated protocol buffer package.

It is generated from these files:
	proto/exsync.proto

It has these top-level messages:
	ReqEmpty
	ReqOrder
	Index
	Orders
	Trades
	Order
	Trade
	Depth
	Balance
	Position
	Account
*/
package exsync

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type ReqEmpty struct {
}

func (m *ReqEmpty) Reset()                    { *m = ReqEmpty{} }
func (m *ReqEmpty) String() string            { return proto.CompactTextString(m) }
func (*ReqEmpty) ProtoMessage()               {}
func (*ReqEmpty) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type ReqOrder struct {
	Ids []int64 `protobuf:"varint,1,rep,packed,name=ids" json:"ids,omitempty"`
}

func (m *ReqOrder) Reset()                    { *m = ReqOrder{} }
func (m *ReqOrder) String() string            { return proto.CompactTextString(m) }
func (*ReqOrder) ProtoMessage()               {}
func (*ReqOrder) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *ReqOrder) GetIds() []int64 {
	if m != nil {
		return m.Ids
	}
	return nil
}

type Index struct {
	Index float64 `protobuf:"fixed64,1,opt,name=index" json:"index,omitempty"`
}

func (m *Index) Reset()                    { *m = Index{} }
func (m *Index) String() string            { return proto.CompactTextString(m) }
func (*Index) ProtoMessage()               {}
func (*Index) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *Index) GetIndex() float64 {
	if m != nil {
		return m.Index
	}
	return 0
}

type Orders struct {
	Orders []*Order `protobuf:"bytes,1,rep,name=orders" json:"orders,omitempty"`
}

func (m *Orders) Reset()                    { *m = Orders{} }
func (m *Orders) String() string            { return proto.CompactTextString(m) }
func (*Orders) ProtoMessage()               {}
func (*Orders) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *Orders) GetOrders() []*Order {
	if m != nil {
		return m.Orders
	}
	return nil
}

type Trades struct {
	Trades []*Trade `protobuf:"bytes,1,rep,name=trades" json:"trades,omitempty"`
}

func (m *Trades) Reset()                    { *m = Trades{} }
func (m *Trades) String() string            { return proto.CompactTextString(m) }
func (*Trades) ProtoMessage()               {}
func (*Trades) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *Trades) GetTrades() []*Trade {
	if m != nil {
		return m.Trades
	}
	return nil
}

type Order struct {
	Id         string  `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	Taction    string  `protobuf:"bytes,2,opt,name=taction" json:"taction,omitempty"`
	Amount     float64 `protobuf:"fixed64,3,opt,name=amount" json:"amount,omitempty"`
	Price      float64 `protobuf:"fixed64,4,opt,name=price" json:"price,omitempty"`
	DealAmount float64 `protobuf:"fixed64,5,opt,name=dealAmount" json:"dealAmount,omitempty"`
	DealMoney  float64 `protobuf:"fixed64,6,opt,name=dealMoney" json:"dealMoney,omitempty"`
	AvgPrice   float64 `protobuf:"fixed64,7,opt,name=avgPrice" json:"avgPrice,omitempty"`
	Fee        float64 `protobuf:"fixed64,8,opt,name=fee" json:"fee,omitempty"`
	Status     int32   `protobuf:"varint,9,opt,name=status" json:"status,omitempty"`
	Timenano   int64   `protobuf:"varint,10,opt,name=timenano" json:"timenano,omitempty"`
}

func (m *Order) Reset()                    { *m = Order{} }
func (m *Order) String() string            { return proto.CompactTextString(m) }
func (*Order) ProtoMessage()               {}
func (*Order) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *Order) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Order) GetTaction() string {
	if m != nil {
		return m.Taction
	}
	return ""
}

func (m *Order) GetAmount() float64 {
	if m != nil {
		return m.Amount
	}
	return 0
}

func (m *Order) GetPrice() float64 {
	if m != nil {
		return m.Price
	}
	return 0
}

func (m *Order) GetDealAmount() float64 {
	if m != nil {
		return m.DealAmount
	}
	return 0
}

func (m *Order) GetDealMoney() float64 {
	if m != nil {
		return m.DealMoney
	}
	return 0
}

func (m *Order) GetAvgPrice() float64 {
	if m != nil {
		return m.AvgPrice
	}
	return 0
}

func (m *Order) GetFee() float64 {
	if m != nil {
		return m.Fee
	}
	return 0
}

func (m *Order) GetStatus() int32 {
	if m != nil {
		return m.Status
	}
	return 0
}

func (m *Order) GetTimenano() int64 {
	if m != nil {
		return m.Timenano
	}
	return 0
}

type Trade struct {
	Id       string  `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	Taction  string  `protobuf:"bytes,2,opt,name=taction" json:"taction,omitempty"`
	Amount   float64 `protobuf:"fixed64,3,opt,name=amount" json:"amount,omitempty"`
	Price    float64 `protobuf:"fixed64,4,opt,name=price" json:"price,omitempty"`
	Fee      float64 `protobuf:"fixed64,5,opt,name=fee" json:"fee,omitempty"`
	Timenano int64   `protobuf:"varint,6,opt,name=timenano" json:"timenano,omitempty"`
}

func (m *Trade) Reset()                    { *m = Trade{} }
func (m *Trade) String() string            { return proto.CompactTextString(m) }
func (*Trade) ProtoMessage()               {}
func (*Trade) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *Trade) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Trade) GetTaction() string {
	if m != nil {
		return m.Taction
	}
	return ""
}

func (m *Trade) GetAmount() float64 {
	if m != nil {
		return m.Amount
	}
	return 0
}

func (m *Trade) GetPrice() float64 {
	if m != nil {
		return m.Price
	}
	return 0
}

func (m *Trade) GetFee() float64 {
	if m != nil {
		return m.Fee
	}
	return 0
}

func (m *Trade) GetTimenano() int64 {
	if m != nil {
		return m.Timenano
	}
	return 0
}

type Depth struct {
	Asks []*Order `protobuf:"bytes,1,rep,name=asks" json:"asks,omitempty"`
	Bids []*Order `protobuf:"bytes,2,rep,name=bids" json:"bids,omitempty"`
}

func (m *Depth) Reset()                    { *m = Depth{} }
func (m *Depth) String() string            { return proto.CompactTextString(m) }
func (*Depth) ProtoMessage()               {}
func (*Depth) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

func (m *Depth) GetAsks() []*Order {
	if m != nil {
		return m.Asks
	}
	return nil
}

func (m *Depth) GetBids() []*Order {
	if m != nil {
		return m.Bids
	}
	return nil
}

type Balance struct {
	Amount       float64 `protobuf:"fixed64,1,opt,name=amount" json:"amount,omitempty"`
	Deposit      float64 `protobuf:"fixed64,2,opt,name=deposit" json:"deposit,omitempty"`
	RealProfil   float64 `protobuf:"fixed64,3,opt,name=realProfil" json:"realProfil,omitempty"`
	UnrealProfit float64 `protobuf:"fixed64,4,opt,name=UnrealProfit,json=unrealProfit" json:"UnrealProfit,omitempty"`
	RiskRate     float64 `protobuf:"fixed64,5,opt,name=RiskRate,json=riskRate" json:"RiskRate,omitempty"`
	Currentcy    string  `protobuf:"bytes,6,opt,name=currentcy" json:"currentcy,omitempty"`
}

func (m *Balance) Reset()                    { *m = Balance{} }
func (m *Balance) String() string            { return proto.CompactTextString(m) }
func (*Balance) ProtoMessage()               {}
func (*Balance) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{8} }

func (m *Balance) GetAmount() float64 {
	if m != nil {
		return m.Amount
	}
	return 0
}

func (m *Balance) GetDeposit() float64 {
	if m != nil {
		return m.Deposit
	}
	return 0
}

func (m *Balance) GetRealProfil() float64 {
	if m != nil {
		return m.RealProfil
	}
	return 0
}

func (m *Balance) GetUnrealProfit() float64 {
	if m != nil {
		return m.UnrealProfit
	}
	return 0
}

func (m *Balance) GetRiskRate() float64 {
	if m != nil {
		return m.RiskRate
	}
	return 0
}

func (m *Balance) GetCurrentcy() string {
	if m != nil {
		return m.Currentcy
	}
	return ""
}

type Position struct {
	Id              string  `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	Ptype           string  `protobuf:"bytes,2,opt,name=ptype" json:"ptype,omitempty"`
	Amount          float64 `protobuf:"fixed64,3,opt,name=amount" json:"amount,omitempty"`
	AvailableAmount float64 `protobuf:"fixed64,4,opt,name=availableAmount" json:"availableAmount,omitempty"`
	AvgPrice        float64 `protobuf:"fixed64,5,opt,name=avgPrice" json:"avgPrice,omitempty"`
	Money           float64 `protobuf:"fixed64,6,opt,name=money" json:"money,omitempty"`
	Deposti         float64 `protobuf:"fixed64,7,opt,name=deposti" json:"deposti,omitempty"`
	Leverge         float64 `protobuf:"fixed64,8,opt,name=leverge" json:"leverge,omitempty"`
	ForceClosePrice float64 `protobuf:"fixed64,9,opt,name=forceClosePrice" json:"forceClosePrice,omitempty"`
}

func (m *Position) Reset()                    { *m = Position{} }
func (m *Position) String() string            { return proto.CompactTextString(m) }
func (*Position) ProtoMessage()               {}
func (*Position) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{9} }

func (m *Position) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Position) GetPtype() string {
	if m != nil {
		return m.Ptype
	}
	return ""
}

func (m *Position) GetAmount() float64 {
	if m != nil {
		return m.Amount
	}
	return 0
}

func (m *Position) GetAvailableAmount() float64 {
	if m != nil {
		return m.AvailableAmount
	}
	return 0
}

func (m *Position) GetAvgPrice() float64 {
	if m != nil {
		return m.AvgPrice
	}
	return 0
}

func (m *Position) GetMoney() float64 {
	if m != nil {
		return m.Money
	}
	return 0
}

func (m *Position) GetDeposti() float64 {
	if m != nil {
		return m.Deposti
	}
	return 0
}

func (m *Position) GetLeverge() float64 {
	if m != nil {
		return m.Leverge
	}
	return 0
}

func (m *Position) GetForceClosePrice() float64 {
	if m != nil {
		return m.ForceClosePrice
	}
	return 0
}

type Account struct {
	Balance   *Balance    `protobuf:"bytes,1,opt,name=balance" json:"balance,omitempty"`
	Positions []*Position `protobuf:"bytes,2,rep,name=positions" json:"positions,omitempty"`
}

func (m *Account) Reset()                    { *m = Account{} }
func (m *Account) String() string            { return proto.CompactTextString(m) }
func (*Account) ProtoMessage()               {}
func (*Account) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{10} }

func (m *Account) GetBalance() *Balance {
	if m != nil {
		return m.Balance
	}
	return nil
}

func (m *Account) GetPositions() []*Position {
	if m != nil {
		return m.Positions
	}
	return nil
}

func init() {
	proto.RegisterType((*ReqEmpty)(nil), "exsync.ReqEmpty")
	proto.RegisterType((*ReqOrder)(nil), "exsync.ReqOrder")
	proto.RegisterType((*Index)(nil), "exsync.Index")
	proto.RegisterType((*Orders)(nil), "exsync.Orders")
	proto.RegisterType((*Trades)(nil), "exsync.Trades")
	proto.RegisterType((*Order)(nil), "exsync.Order")
	proto.RegisterType((*Trade)(nil), "exsync.Trade")
	proto.RegisterType((*Depth)(nil), "exsync.Depth")
	proto.RegisterType((*Balance)(nil), "exsync.Balance")
	proto.RegisterType((*Position)(nil), "exsync.Position")
	proto.RegisterType((*Account)(nil), "exsync.Account")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for SyncService service

type SyncServiceClient interface {
	GetOrders(ctx context.Context, in *ReqOrder, opts ...grpc.CallOption) (*Orders, error)
	GetTrades(ctx context.Context, in *ReqEmpty, opts ...grpc.CallOption) (*Trades, error)
	GetDepth(ctx context.Context, in *ReqEmpty, opts ...grpc.CallOption) (*Depth, error)
	GetIndex(ctx context.Context, in *ReqEmpty, opts ...grpc.CallOption) (*Index, error)
	GetAccount(ctx context.Context, in *ReqEmpty, opts ...grpc.CallOption) (*Account, error)
}

type syncServiceClient struct {
	cc *grpc.ClientConn
}

func NewSyncServiceClient(cc *grpc.ClientConn) SyncServiceClient {
	return &syncServiceClient{cc}
}

func (c *syncServiceClient) GetOrders(ctx context.Context, in *ReqOrder, opts ...grpc.CallOption) (*Orders, error) {
	out := new(Orders)
	err := grpc.Invoke(ctx, "/exsync.SyncService/GetOrders", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *syncServiceClient) GetTrades(ctx context.Context, in *ReqEmpty, opts ...grpc.CallOption) (*Trades, error) {
	out := new(Trades)
	err := grpc.Invoke(ctx, "/exsync.SyncService/GetTrades", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *syncServiceClient) GetDepth(ctx context.Context, in *ReqEmpty, opts ...grpc.CallOption) (*Depth, error) {
	out := new(Depth)
	err := grpc.Invoke(ctx, "/exsync.SyncService/GetDepth", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *syncServiceClient) GetIndex(ctx context.Context, in *ReqEmpty, opts ...grpc.CallOption) (*Index, error) {
	out := new(Index)
	err := grpc.Invoke(ctx, "/exsync.SyncService/GetIndex", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *syncServiceClient) GetAccount(ctx context.Context, in *ReqEmpty, opts ...grpc.CallOption) (*Account, error) {
	out := new(Account)
	err := grpc.Invoke(ctx, "/exsync.SyncService/GetAccount", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for SyncService service

type SyncServiceServer interface {
	GetOrders(context.Context, *ReqOrder) (*Orders, error)
	GetTrades(context.Context, *ReqEmpty) (*Trades, error)
	GetDepth(context.Context, *ReqEmpty) (*Depth, error)
	GetIndex(context.Context, *ReqEmpty) (*Index, error)
	GetAccount(context.Context, *ReqEmpty) (*Account, error)
}

func RegisterSyncServiceServer(s *grpc.Server, srv SyncServiceServer) {
	s.RegisterService(&_SyncService_serviceDesc, srv)
}

func _SyncService_GetOrders_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReqOrder)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SyncServiceServer).GetOrders(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/exsync.SyncService/GetOrders",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SyncServiceServer).GetOrders(ctx, req.(*ReqOrder))
	}
	return interceptor(ctx, in, info, handler)
}

func _SyncService_GetTrades_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReqEmpty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SyncServiceServer).GetTrades(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/exsync.SyncService/GetTrades",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SyncServiceServer).GetTrades(ctx, req.(*ReqEmpty))
	}
	return interceptor(ctx, in, info, handler)
}

func _SyncService_GetDepth_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReqEmpty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SyncServiceServer).GetDepth(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/exsync.SyncService/GetDepth",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SyncServiceServer).GetDepth(ctx, req.(*ReqEmpty))
	}
	return interceptor(ctx, in, info, handler)
}

func _SyncService_GetIndex_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReqEmpty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SyncServiceServer).GetIndex(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/exsync.SyncService/GetIndex",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SyncServiceServer).GetIndex(ctx, req.(*ReqEmpty))
	}
	return interceptor(ctx, in, info, handler)
}

func _SyncService_GetAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReqEmpty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SyncServiceServer).GetAccount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/exsync.SyncService/GetAccount",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SyncServiceServer).GetAccount(ctx, req.(*ReqEmpty))
	}
	return interceptor(ctx, in, info, handler)
}

var _SyncService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "exsync.SyncService",
	HandlerType: (*SyncServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetOrders",
			Handler:    _SyncService_GetOrders_Handler,
		},
		{
			MethodName: "GetTrades",
			Handler:    _SyncService_GetTrades_Handler,
		},
		{
			MethodName: "GetDepth",
			Handler:    _SyncService_GetDepth_Handler,
		},
		{
			MethodName: "GetIndex",
			Handler:    _SyncService_GetIndex_Handler,
		},
		{
			MethodName: "GetAccount",
			Handler:    _SyncService_GetAccount_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/exsync.proto",
}

func init() { proto.RegisterFile("proto/exsync.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 629 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xb4, 0x54, 0xdd, 0x6e, 0xd3, 0x4c,
	0x10, 0x8d, 0x93, 0xda, 0x89, 0xa7, 0xfd, 0xda, 0x6a, 0x55, 0x7d, 0xb2, 0xaa, 0x82, 0xc2, 0x4a,
	0x48, 0xe1, 0x82, 0x56, 0x94, 0x27, 0x28, 0x3f, 0xaa, 0xb8, 0xa8, 0xa8, 0xb6, 0xf0, 0x00, 0x1b,
	0x7b, 0x5a, 0xac, 0x3a, 0xb6, 0x59, 0x6f, 0xa2, 0xe6, 0x0e, 0x71, 0xcf, 0x03, 0xf1, 0x74, 0xa0,
	0x9d, 0xdd, 0x75, 0x5c, 0x97, 0x72, 0xc7, 0xdd, 0x9e, 0xf9, 0xb1, 0xcf, 0x9c, 0x39, 0x1a, 0x60,
	0xb5, 0xaa, 0x74, 0x75, 0x82, 0x77, 0xcd, 0xba, 0x4c, 0x8f, 0x09, 0xb0, 0xc8, 0x22, 0x0e, 0x30,
	0x11, 0xf8, 0xf5, 0xfd, 0xa2, 0xd6, 0x6b, 0x7e, 0x44, 0xef, 0x8f, 0x2a, 0x43, 0xc5, 0xf6, 0x61,
	0x94, 0x67, 0x4d, 0x12, 0x4c, 0x47, 0xb3, 0x91, 0x30, 0x4f, 0xfe, 0x04, 0xc2, 0x0f, 0x65, 0x86,
	0x77, 0xec, 0x00, 0xc2, 0xdc, 0x3c, 0x92, 0x60, 0x1a, 0xcc, 0x02, 0x61, 0x01, 0x3f, 0x81, 0x88,
	0x3a, 0x1b, 0xf6, 0x1c, 0xa2, 0x8a, 0x5e, 0xd4, 0xbd, 0x7d, 0xfa, 0xdf, 0xb1, 0xfb, 0x33, 0xe5,
	0x85, 0x4b, 0x9a, 0x86, 0x4f, 0x4a, 0x66, 0x48, 0x0d, 0x9a, 0x5e, 0xfd, 0x06, 0xca, 0x0b, 0x97,
	0xe4, 0xbf, 0x02, 0x08, 0x2d, 0xb9, 0x5d, 0x18, 0xe6, 0x19, 0xfd, 0x3e, 0x16, 0xc3, 0x3c, 0x63,
	0x09, 0x8c, 0xb5, 0x4c, 0x75, 0x5e, 0x95, 0xc9, 0x90, 0x82, 0x1e, 0xb2, 0xff, 0x21, 0x92, 0x8b,
	0x6a, 0x59, 0xea, 0x64, 0x44, 0x64, 0x1d, 0x32, 0x33, 0xd4, 0x2a, 0x4f, 0x31, 0xd9, 0xb2, 0x33,
	0x10, 0x60, 0x4f, 0x01, 0x32, 0x94, 0xc5, 0x99, 0xed, 0x08, 0x29, 0xd5, 0x89, 0xb0, 0x23, 0x88,
	0x0d, 0xba, 0xa8, 0x4a, 0x5c, 0x27, 0x11, 0xa5, 0x37, 0x01, 0x76, 0x08, 0x13, 0xb9, 0xba, 0xb9,
	0xa4, 0xcf, 0x8e, 0x29, 0xd9, 0x62, 0x23, 0xe7, 0x35, 0x62, 0x32, 0xa1, 0xb0, 0x79, 0x1a, 0x66,
	0x8d, 0x96, 0x7a, 0xd9, 0x24, 0xf1, 0x34, 0x98, 0x85, 0xc2, 0x21, 0xf3, 0x15, 0x9d, 0x2f, 0xb0,
	0x94, 0x65, 0x95, 0xc0, 0x34, 0x98, 0x8d, 0x44, 0x8b, 0xf9, 0x8f, 0x00, 0x42, 0xd2, 0xe4, 0x9f,
	0x29, 0xe0, 0x78, 0x86, 0x1b, 0x9e, 0x5d, 0x3e, 0x51, 0x8f, 0xcf, 0x05, 0x84, 0xef, 0xb0, 0xd6,
	0x5f, 0xd8, 0x33, 0xd8, 0x92, 0xcd, 0xed, 0x23, 0x0b, 0xa7, 0x94, 0x29, 0x99, 0x1b, 0x47, 0x0d,
	0xff, 0x58, 0x62, 0x52, 0xfc, 0x67, 0x00, 0xe3, 0x37, 0xb2, 0x90, 0x65, 0x8a, 0x1d, 0xda, 0xc1,
	0x3d, 0xda, 0x09, 0x8c, 0x33, 0xac, 0xab, 0x26, 0xd7, 0x34, 0x68, 0x20, 0x3c, 0x34, 0xcb, 0x53,
	0x28, 0x8b, 0x4b, 0x55, 0x5d, 0xe7, 0x85, 0x1b, 0xb6, 0x13, 0x61, 0x1c, 0x76, 0x3e, 0x97, 0x2d,
	0xd6, 0x6e, 0xee, 0x9d, 0x65, 0x27, 0x66, 0x86, 0x15, 0x79, 0x73, 0x2b, 0xa4, 0xf6, 0x1a, 0x4c,
	0x94, 0xc3, 0x66, 0xf9, 0xe9, 0x52, 0x29, 0x2c, 0x75, 0x6a, 0x97, 0x1f, 0x8b, 0x4d, 0x80, 0x7f,
	0x1f, 0xc2, 0xe4, 0xd2, 0xf0, 0x30, 0x9a, 0xf7, 0xb7, 0x63, 0xb4, 0xd6, 0xeb, 0x1a, 0xdd, 0x6e,
	0x2c, 0x78, 0x74, 0x33, 0x33, 0xd8, 0x93, 0x2b, 0x99, 0x17, 0x72, 0x5e, 0xa0, 0xb3, 0xa2, 0xe5,
	0xda, 0x0f, 0xdf, 0x73, 0x5c, 0xd8, 0x73, 0xdc, 0x01, 0x84, 0x8b, 0x8e, 0x4f, 0x2d, 0x68, 0xe5,
	0xd3, 0xb9, 0xb3, 0xa8, 0x87, 0x26, 0x53, 0xe0, 0x0a, 0xd5, 0x8d, 0x77, 0xa9, 0x87, 0x86, 0xcf,
	0x75, 0xa5, 0x52, 0x7c, 0x5b, 0x54, 0x0d, 0xda, 0x9f, 0xc5, 0x96, 0x4f, 0x2f, 0xcc, 0x33, 0x18,
	0x9f, 0xa5, 0x29, 0x51, 0x7b, 0x01, 0xe3, 0xb9, 0x5d, 0x25, 0xe9, 0xb0, 0x7d, 0xba, 0xe7, 0x37,
	0xee, 0x36, 0x2c, 0x7c, 0x9e, 0x1d, 0x43, 0x5c, 0x3b, 0xe5, 0xbc, 0x3d, 0xf6, 0x7d, 0xb1, 0x97,
	0x54, 0x6c, 0x4a, 0x4e, 0xbf, 0x0d, 0x61, 0xfb, 0x6a, 0x5d, 0xa6, 0x57, 0xa8, 0x56, 0x66, 0xd2,
	0x13, 0x88, 0xcf, 0x51, 0xbb, 0xe3, 0xd3, 0x76, 0xfa, 0x4b, 0x76, 0xb8, 0x7b, 0xcf, 0x6a, 0x0d,
	0x1f, 0xb8, 0x06, 0x77, 0x7c, 0xba, 0x0d, 0x74, 0x06, 0x37, 0x0d, 0xb6, 0x82, 0x0f, 0xd8, 0x4b,
	0x98, 0x9c, 0xa3, 0xb6, 0x56, 0x7f, 0x58, 0xdf, 0x7a, 0x99, 0x0a, 0xda, 0x72, 0x7b, 0x2c, 0xff,
	0x52, 0x4e, 0x05, 0x7c, 0xc0, 0x5e, 0x01, 0x9c, 0xa3, 0xf6, 0xc2, 0x3d, 0x6c, 0x68, 0x95, 0x73,
	0x25, 0x7c, 0x30, 0x8f, 0xe8, 0x88, 0xbf, 0xfe, 0x1d, 0x00, 0x00, 0xff, 0xff, 0x9f, 0x44, 0x91,
	0x21, 0xda, 0x05, 0x00, 0x00,
}