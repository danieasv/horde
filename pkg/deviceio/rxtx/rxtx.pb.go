// Code generated by protoc-gen-go. DO NOT EDIT.
// source: rxtx.proto

package rxtx

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

// Message types. CoAPPull is device-initated downstream messages, CoAP push is
// horde-initiated downstream messages and CoAP upstream is general CoAP
// upstream messages. Technically they're CoAP pull messages but it makes it
// easier to follow the logic when there's three kinds of coap messages.
// The UDP messages goes both ways; context determines wether it's upstream
// or downstream.
type MessageType int32

const (
	MessageType_UDP          MessageType = 0
	MessageType_CoAPUpstream MessageType = 1
	MessageType_CoAPPull     MessageType = 2
	MessageType_CoAPPush     MessageType = 3
	MessageType_UDPPull      MessageType = 4
)

var MessageType_name = map[int32]string{
	0: "UDP",
	1: "CoAPUpstream",
	2: "CoAPPull",
	3: "CoAPPush",
	4: "UDPPull",
}

var MessageType_value = map[string]int32{
	"UDP":          0,
	"CoAPUpstream": 1,
	"CoAPPull":     2,
	"CoAPPush":     3,
	"UDPPull":      4,
}

func (x MessageType) String() string {
	return proto.EnumName(MessageType_name, int32(x))
}

func (MessageType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_718277bfb8eee15a, []int{0}
}

type ErrorCode int32

const (
	ErrorCode_SUCCESS      ErrorCode = 0
	ErrorCode_TOO_LARGE    ErrorCode = 1
	ErrorCode_NETWORK      ErrorCode = 2
	ErrorCode_NOT_HANDLED  ErrorCode = 3
	ErrorCode_CLIENT_ERROR ErrorCode = 4
	ErrorCode_PARAMETER    ErrorCode = 5
	ErrorCode_INTERNAL     ErrorCode = 6
	ErrorCode_PENDING      ErrorCode = 7
	ErrorCode_TIMEOUT      ErrorCode = 8
)

var ErrorCode_name = map[int32]string{
	0: "SUCCESS",
	1: "TOO_LARGE",
	2: "NETWORK",
	3: "NOT_HANDLED",
	4: "CLIENT_ERROR",
	5: "PARAMETER",
	6: "INTERNAL",
	7: "PENDING",
	8: "TIMEOUT",
}

var ErrorCode_value = map[string]int32{
	"SUCCESS":      0,
	"TOO_LARGE":    1,
	"NETWORK":      2,
	"NOT_HANDLED":  3,
	"CLIENT_ERROR": 4,
	"PARAMETER":    5,
	"INTERNAL":     6,
	"PENDING":      7,
	"TIMEOUT":      8,
}

func (x ErrorCode) String() string {
	return proto.EnumName(ErrorCode_name, int32(x))
}

func (ErrorCode) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_718277bfb8eee15a, []int{1}
}

type UDPOptions struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *UDPOptions) Reset()         { *m = UDPOptions{} }
func (m *UDPOptions) String() string { return proto.CompactTextString(m) }
func (*UDPOptions) ProtoMessage()    {}
func (*UDPOptions) Descriptor() ([]byte, []int) {
	return fileDescriptor_718277bfb8eee15a, []int{0}
}

func (m *UDPOptions) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UDPOptions.Unmarshal(m, b)
}
func (m *UDPOptions) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UDPOptions.Marshal(b, m, deterministic)
}
func (m *UDPOptions) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UDPOptions.Merge(m, src)
}
func (m *UDPOptions) XXX_Size() int {
	return xxx_messageInfo_UDPOptions.Size(m)
}
func (m *UDPOptions) XXX_DiscardUnknown() {
	xxx_messageInfo_UDPOptions.DiscardUnknown(m)
}

var xxx_messageInfo_UDPOptions proto.InternalMessageInfo

type CoAPOptions struct {
	Code                 int32    `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Type                 int32    `protobuf:"varint,2,opt,name=type,proto3" json:"type,omitempty"`
	LocationPath         []string `protobuf:"bytes,3,rep,name=location_path,json=locationPath,proto3" json:"location_path,omitempty"`
	Path                 string   `protobuf:"bytes,4,opt,name=path,proto3" json:"path,omitempty"`
	ContentFormat        int32    `protobuf:"varint,5,opt,name=content_format,json=contentFormat,proto3" json:"content_format,omitempty"`
	UriQuery             []string `protobuf:"bytes,6,rep,name=uri_query,json=uriQuery,proto3" json:"uri_query,omitempty"`
	Accept               int32    `protobuf:"varint,7,opt,name=accept,proto3" json:"accept,omitempty"`
	Token                int64    `protobuf:"varint,9,opt,name=token,proto3" json:"token,omitempty"`
	TimeoutSeconds       int32    `protobuf:"varint,10,opt,name=timeout_seconds,json=timeoutSeconds,proto3" json:"timeout_seconds,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CoAPOptions) Reset()         { *m = CoAPOptions{} }
func (m *CoAPOptions) String() string { return proto.CompactTextString(m) }
func (*CoAPOptions) ProtoMessage()    {}
func (*CoAPOptions) Descriptor() ([]byte, []int) {
	return fileDescriptor_718277bfb8eee15a, []int{1}
}

func (m *CoAPOptions) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CoAPOptions.Unmarshal(m, b)
}
func (m *CoAPOptions) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CoAPOptions.Marshal(b, m, deterministic)
}
func (m *CoAPOptions) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CoAPOptions.Merge(m, src)
}
func (m *CoAPOptions) XXX_Size() int {
	return xxx_messageInfo_CoAPOptions.Size(m)
}
func (m *CoAPOptions) XXX_DiscardUnknown() {
	xxx_messageInfo_CoAPOptions.DiscardUnknown(m)
}

var xxx_messageInfo_CoAPOptions proto.InternalMessageInfo

func (m *CoAPOptions) GetCode() int32 {
	if m != nil {
		return m.Code
	}
	return 0
}

func (m *CoAPOptions) GetType() int32 {
	if m != nil {
		return m.Type
	}
	return 0
}

func (m *CoAPOptions) GetLocationPath() []string {
	if m != nil {
		return m.LocationPath
	}
	return nil
}

func (m *CoAPOptions) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

func (m *CoAPOptions) GetContentFormat() int32 {
	if m != nil {
		return m.ContentFormat
	}
	return 0
}

func (m *CoAPOptions) GetUriQuery() []string {
	if m != nil {
		return m.UriQuery
	}
	return nil
}

func (m *CoAPOptions) GetAccept() int32 {
	if m != nil {
		return m.Accept
	}
	return 0
}

func (m *CoAPOptions) GetToken() int64 {
	if m != nil {
		return m.Token
	}
	return 0
}

func (m *CoAPOptions) GetTimeoutSeconds() int32 {
	if m != nil {
		return m.TimeoutSeconds
	}
	return 0
}

type Message struct {
	Id                   int64        `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Type                 MessageType  `protobuf:"varint,2,opt,name=type,proto3,enum=rxtx.MessageType" json:"type,omitempty"`
	Timestamp            int64        `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	RemoteAddress        []byte       `protobuf:"bytes,4,opt,name=remote_address,json=remoteAddress,proto3" json:"remote_address,omitempty"`
	RemotePort           int32        `protobuf:"varint,5,opt,name=remote_port,json=remotePort,proto3" json:"remote_port,omitempty"`
	LocalPort            int32        `protobuf:"varint,6,opt,name=local_port,json=localPort,proto3" json:"local_port,omitempty"`
	Payload              []byte       `protobuf:"bytes,7,opt,name=payload,proto3" json:"payload,omitempty"`
	Coap                 *CoAPOptions `protobuf:"bytes,8,opt,name=coap,proto3" json:"coap,omitempty"`
	Udp                  *UDPOptions  `protobuf:"bytes,9,opt,name=udp,proto3" json:"udp,omitempty"`
	XXX_NoUnkeyedLiteral struct{}     `json:"-"`
	XXX_unrecognized     []byte       `json:"-"`
	XXX_sizecache        int32        `json:"-"`
}

func (m *Message) Reset()         { *m = Message{} }
func (m *Message) String() string { return proto.CompactTextString(m) }
func (*Message) ProtoMessage()    {}
func (*Message) Descriptor() ([]byte, []int) {
	return fileDescriptor_718277bfb8eee15a, []int{2}
}

func (m *Message) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Message.Unmarshal(m, b)
}
func (m *Message) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Message.Marshal(b, m, deterministic)
}
func (m *Message) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Message.Merge(m, src)
}
func (m *Message) XXX_Size() int {
	return xxx_messageInfo_Message.Size(m)
}
func (m *Message) XXX_DiscardUnknown() {
	xxx_messageInfo_Message.DiscardUnknown(m)
}

var xxx_messageInfo_Message proto.InternalMessageInfo

func (m *Message) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *Message) GetType() MessageType {
	if m != nil {
		return m.Type
	}
	return MessageType_UDP
}

func (m *Message) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *Message) GetRemoteAddress() []byte {
	if m != nil {
		return m.RemoteAddress
	}
	return nil
}

func (m *Message) GetRemotePort() int32 {
	if m != nil {
		return m.RemotePort
	}
	return 0
}

func (m *Message) GetLocalPort() int32 {
	if m != nil {
		return m.LocalPort
	}
	return 0
}

func (m *Message) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *Message) GetCoap() *CoAPOptions {
	if m != nil {
		return m.Coap
	}
	return nil
}

func (m *Message) GetUdp() *UDPOptions {
	if m != nil {
		return m.Udp
	}
	return nil
}

// Origin tells the server where the request has originated. The APN ID must be
// set. The NAS ID is optional and can be set to -1 if it does not apply. If the
// listener is capable of routing messages to the entire APN the NAS ID can be
// omitted.
type Origin struct {
	ApnId                int32    `protobuf:"varint,1,opt,name=apn_id,json=apnId,proto3" json:"apn_id,omitempty"`
	NasId                []int32  `protobuf:"varint,2,rep,packed,name=nas_id,json=nasId,proto3" json:"nas_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Origin) Reset()         { *m = Origin{} }
func (m *Origin) String() string { return proto.CompactTextString(m) }
func (*Origin) ProtoMessage()    {}
func (*Origin) Descriptor() ([]byte, []int) {
	return fileDescriptor_718277bfb8eee15a, []int{3}
}

func (m *Origin) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Origin.Unmarshal(m, b)
}
func (m *Origin) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Origin.Marshal(b, m, deterministic)
}
func (m *Origin) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Origin.Merge(m, src)
}
func (m *Origin) XXX_Size() int {
	return xxx_messageInfo_Origin.Size(m)
}
func (m *Origin) XXX_DiscardUnknown() {
	xxx_messageInfo_Origin.DiscardUnknown(m)
}

var xxx_messageInfo_Origin proto.InternalMessageInfo

func (m *Origin) GetApnId() int32 {
	if m != nil {
		return m.ApnId
	}
	return 0
}

func (m *Origin) GetNasId() []int32 {
	if m != nil {
		return m.NasId
	}
	return nil
}

// The upstream request is sent by the listener when upstream data (or a request
// is received)
type UpstreamRequest struct {
	Origin               *Origin  `protobuf:"bytes,1,opt,name=origin,proto3" json:"origin,omitempty"`
	Redelivery           bool     `protobuf:"varint,2,opt,name=redelivery,proto3" json:"redelivery,omitempty"`
	Msg                  *Message `protobuf:"bytes,5,opt,name=msg,proto3" json:"msg,omitempty"`
	ExpectDownstream     bool     `protobuf:"varint,6,opt,name=expect_downstream,json=expectDownstream,proto3" json:"expect_downstream,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *UpstreamRequest) Reset()         { *m = UpstreamRequest{} }
func (m *UpstreamRequest) String() string { return proto.CompactTextString(m) }
func (*UpstreamRequest) ProtoMessage()    {}
func (*UpstreamRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_718277bfb8eee15a, []int{4}
}

func (m *UpstreamRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UpstreamRequest.Unmarshal(m, b)
}
func (m *UpstreamRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UpstreamRequest.Marshal(b, m, deterministic)
}
func (m *UpstreamRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UpstreamRequest.Merge(m, src)
}
func (m *UpstreamRequest) XXX_Size() int {
	return xxx_messageInfo_UpstreamRequest.Size(m)
}
func (m *UpstreamRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_UpstreamRequest.DiscardUnknown(m)
}

var xxx_messageInfo_UpstreamRequest proto.InternalMessageInfo

func (m *UpstreamRequest) GetOrigin() *Origin {
	if m != nil {
		return m.Origin
	}
	return nil
}

func (m *UpstreamRequest) GetRedelivery() bool {
	if m != nil {
		return m.Redelivery
	}
	return false
}

func (m *UpstreamRequest) GetMsg() *Message {
	if m != nil {
		return m.Msg
	}
	return nil
}

func (m *UpstreamRequest) GetExpectDownstream() bool {
	if m != nil {
		return m.ExpectDownstream
	}
	return false
}

//
type DownstreamResponse struct {
	Msg                  *Message `protobuf:"bytes,2,opt,name=msg,proto3" json:"msg,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DownstreamResponse) Reset()         { *m = DownstreamResponse{} }
func (m *DownstreamResponse) String() string { return proto.CompactTextString(m) }
func (*DownstreamResponse) ProtoMessage()    {}
func (*DownstreamResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_718277bfb8eee15a, []int{5}
}

func (m *DownstreamResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DownstreamResponse.Unmarshal(m, b)
}
func (m *DownstreamResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DownstreamResponse.Marshal(b, m, deterministic)
}
func (m *DownstreamResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DownstreamResponse.Merge(m, src)
}
func (m *DownstreamResponse) XXX_Size() int {
	return xxx_messageInfo_DownstreamResponse.Size(m)
}
func (m *DownstreamResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_DownstreamResponse.DiscardUnknown(m)
}

var xxx_messageInfo_DownstreamResponse proto.InternalMessageInfo

func (m *DownstreamResponse) GetMsg() *Message {
	if m != nil {
		return m.Msg
	}
	return nil
}

// DownstreamRequest polls for
type DownstreamRequest struct {
	Origin               *Origin     `protobuf:"bytes,1,opt,name=origin,proto3" json:"origin,omitempty"`
	Type                 MessageType `protobuf:"varint,2,opt,name=type,proto3,enum=rxtx.MessageType" json:"type,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *DownstreamRequest) Reset()         { *m = DownstreamRequest{} }
func (m *DownstreamRequest) String() string { return proto.CompactTextString(m) }
func (*DownstreamRequest) ProtoMessage()    {}
func (*DownstreamRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_718277bfb8eee15a, []int{6}
}

func (m *DownstreamRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DownstreamRequest.Unmarshal(m, b)
}
func (m *DownstreamRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DownstreamRequest.Marshal(b, m, deterministic)
}
func (m *DownstreamRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DownstreamRequest.Merge(m, src)
}
func (m *DownstreamRequest) XXX_Size() int {
	return xxx_messageInfo_DownstreamRequest.Size(m)
}
func (m *DownstreamRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DownstreamRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DownstreamRequest proto.InternalMessageInfo

func (m *DownstreamRequest) GetOrigin() *Origin {
	if m != nil {
		return m.Origin
	}
	return nil
}

func (m *DownstreamRequest) GetType() MessageType {
	if m != nil {
		return m.Type
	}
	return MessageType_UDP
}

// The AckRequest message is sent by the listener to the upstream server to ack
// or report errors. A missing result field is interpreted as success.
type AckRequest struct {
	MessageId            int64     `protobuf:"varint,1,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	Result               ErrorCode `protobuf:"varint,2,opt,name=Result,proto3,enum=rxtx.ErrorCode" json:"Result,omitempty"`
	CoapToken            int64     `protobuf:"varint,3,opt,name=coap_token,json=coapToken,proto3" json:"coap_token,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *AckRequest) Reset()         { *m = AckRequest{} }
func (m *AckRequest) String() string { return proto.CompactTextString(m) }
func (*AckRequest) ProtoMessage()    {}
func (*AckRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_718277bfb8eee15a, []int{7}
}

func (m *AckRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AckRequest.Unmarshal(m, b)
}
func (m *AckRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AckRequest.Marshal(b, m, deterministic)
}
func (m *AckRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AckRequest.Merge(m, src)
}
func (m *AckRequest) XXX_Size() int {
	return xxx_messageInfo_AckRequest.Size(m)
}
func (m *AckRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_AckRequest.DiscardUnknown(m)
}

var xxx_messageInfo_AckRequest proto.InternalMessageInfo

func (m *AckRequest) GetMessageId() int64 {
	if m != nil {
		return m.MessageId
	}
	return 0
}

func (m *AckRequest) GetResult() ErrorCode {
	if m != nil {
		return m.Result
	}
	return ErrorCode_SUCCESS
}

func (m *AckRequest) GetCoapToken() int64 {
	if m != nil {
		return m.CoapToken
	}
	return 0
}

// The AckResponse is sent back to the listener.
type AckResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AckResponse) Reset()         { *m = AckResponse{} }
func (m *AckResponse) String() string { return proto.CompactTextString(m) }
func (*AckResponse) ProtoMessage()    {}
func (*AckResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_718277bfb8eee15a, []int{8}
}

func (m *AckResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AckResponse.Unmarshal(m, b)
}
func (m *AckResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AckResponse.Marshal(b, m, deterministic)
}
func (m *AckResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AckResponse.Merge(m, src)
}
func (m *AckResponse) XXX_Size() int {
	return xxx_messageInfo_AckResponse.Size(m)
}
func (m *AckResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_AckResponse.DiscardUnknown(m)
}

var xxx_messageInfo_AckResponse proto.InternalMessageInfo

// AccessRequest is sent from the gRPC-backed RADIUS server to check if
// devices should be allowed to connect.
type AccessRequest struct {
	Imsi                 int64    `protobuf:"varint,1,opt,name=imsi,proto3" json:"imsi,omitempty"`
	NasIdentifier        string   `protobuf:"bytes,2,opt,name=nas_identifier,json=nasIdentifier,proto3" json:"nas_identifier,omitempty"`
	Username             string   `protobuf:"bytes,3,opt,name=username,proto3" json:"username,omitempty"`
	Password             []byte   `protobuf:"bytes,4,opt,name=password,proto3" json:"password,omitempty"`
	UserLocationInfo     []byte   `protobuf:"bytes,5,opt,name=user_location_info,json=userLocationInfo,proto3" json:"user_location_info,omitempty"`
	ImsiMccMnc           string   `protobuf:"bytes,6,opt,name=imsi_mcc_mnc,json=imsiMccMnc,proto3" json:"imsi_mcc_mnc,omitempty"`
	MsTimezone           []byte   `protobuf:"bytes,7,opt,name=ms_timezone,json=msTimezone,proto3" json:"ms_timezone,omitempty"`
	Imeisv               string   `protobuf:"bytes,8,opt,name=imeisv,proto3" json:"imeisv,omitempty"`
	NasIpAddress         []byte   `protobuf:"bytes,9,opt,name=nas_ip_address,json=nasIpAddress,proto3" json:"nas_ip_address,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AccessRequest) Reset()         { *m = AccessRequest{} }
func (m *AccessRequest) String() string { return proto.CompactTextString(m) }
func (*AccessRequest) ProtoMessage()    {}
func (*AccessRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_718277bfb8eee15a, []int{9}
}

func (m *AccessRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AccessRequest.Unmarshal(m, b)
}
func (m *AccessRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AccessRequest.Marshal(b, m, deterministic)
}
func (m *AccessRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AccessRequest.Merge(m, src)
}
func (m *AccessRequest) XXX_Size() int {
	return xxx_messageInfo_AccessRequest.Size(m)
}
func (m *AccessRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_AccessRequest.DiscardUnknown(m)
}

var xxx_messageInfo_AccessRequest proto.InternalMessageInfo

func (m *AccessRequest) GetImsi() int64 {
	if m != nil {
		return m.Imsi
	}
	return 0
}

func (m *AccessRequest) GetNasIdentifier() string {
	if m != nil {
		return m.NasIdentifier
	}
	return ""
}

func (m *AccessRequest) GetUsername() string {
	if m != nil {
		return m.Username
	}
	return ""
}

func (m *AccessRequest) GetPassword() []byte {
	if m != nil {
		return m.Password
	}
	return nil
}

func (m *AccessRequest) GetUserLocationInfo() []byte {
	if m != nil {
		return m.UserLocationInfo
	}
	return nil
}

func (m *AccessRequest) GetImsiMccMnc() string {
	if m != nil {
		return m.ImsiMccMnc
	}
	return ""
}

func (m *AccessRequest) GetMsTimezone() []byte {
	if m != nil {
		return m.MsTimezone
	}
	return nil
}

func (m *AccessRequest) GetImeisv() string {
	if m != nil {
		return m.Imeisv
	}
	return ""
}

func (m *AccessRequest) GetNasIpAddress() []byte {
	if m != nil {
		return m.NasIpAddress
	}
	return nil
}

// AccessResponse is the response to the gRPC-backed RADIUS server.
type AccessResponse struct {
	Accepted             bool     `protobuf:"varint,1,opt,name=accepted,proto3" json:"accepted,omitempty"`
	IpAddress            []byte   `protobuf:"bytes,2,opt,name=ip_address,json=ipAddress,proto3" json:"ip_address,omitempty"`
	Message              string   `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AccessResponse) Reset()         { *m = AccessResponse{} }
func (m *AccessResponse) String() string { return proto.CompactTextString(m) }
func (*AccessResponse) ProtoMessage()    {}
func (*AccessResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_718277bfb8eee15a, []int{10}
}

func (m *AccessResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AccessResponse.Unmarshal(m, b)
}
func (m *AccessResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AccessResponse.Marshal(b, m, deterministic)
}
func (m *AccessResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AccessResponse.Merge(m, src)
}
func (m *AccessResponse) XXX_Size() int {
	return xxx_messageInfo_AccessResponse.Size(m)
}
func (m *AccessResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_AccessResponse.DiscardUnknown(m)
}

var xxx_messageInfo_AccessResponse proto.InternalMessageInfo

func (m *AccessResponse) GetAccepted() bool {
	if m != nil {
		return m.Accepted
	}
	return false
}

func (m *AccessResponse) GetIpAddress() []byte {
	if m != nil {
		return m.IpAddress
	}
	return nil
}

func (m *AccessResponse) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func init() {
	proto.RegisterEnum("rxtx.MessageType", MessageType_name, MessageType_value)
	proto.RegisterEnum("rxtx.ErrorCode", ErrorCode_name, ErrorCode_value)
	proto.RegisterType((*UDPOptions)(nil), "rxtx.UDPOptions")
	proto.RegisterType((*CoAPOptions)(nil), "rxtx.CoAPOptions")
	proto.RegisterType((*Message)(nil), "rxtx.Message")
	proto.RegisterType((*Origin)(nil), "rxtx.Origin")
	proto.RegisterType((*UpstreamRequest)(nil), "rxtx.UpstreamRequest")
	proto.RegisterType((*DownstreamResponse)(nil), "rxtx.DownstreamResponse")
	proto.RegisterType((*DownstreamRequest)(nil), "rxtx.DownstreamRequest")
	proto.RegisterType((*AckRequest)(nil), "rxtx.AckRequest")
	proto.RegisterType((*AckResponse)(nil), "rxtx.AckResponse")
	proto.RegisterType((*AccessRequest)(nil), "rxtx.AccessRequest")
	proto.RegisterType((*AccessResponse)(nil), "rxtx.AccessResponse")
}

func init() { proto.RegisterFile("rxtx.proto", fileDescriptor_718277bfb8eee15a) }

var fileDescriptor_718277bfb8eee15a = []byte{
	// 1024 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x55, 0x41, 0x73, 0xdb, 0x44,
	0x14, 0xae, 0x2d, 0x5b, 0xb6, 0x9e, 0xed, 0x44, 0x59, 0x5a, 0xd0, 0x04, 0x4a, 0x3d, 0xa2, 0x9d,
	0x66, 0x0a, 0xd3, 0x83, 0x3b, 0x70, 0xeb, 0x30, 0x9a, 0x58, 0x04, 0x0f, 0x89, 0xed, 0x6e, 0xe4,
	0xe1, 0x28, 0x84, 0xb4, 0x49, 0x34, 0xb1, 0xb4, 0xaa, 0x76, 0xdd, 0x26, 0xfc, 0x03, 0x2e, 0xfc,
	0x0b, 0x2e, 0xdc, 0xe0, 0x17, 0x32, 0x6f, 0x77, 0xe5, 0x38, 0x03, 0x0c, 0x70, 0xd3, 0xfb, 0xbe,
	0x7d, 0x6f, 0x77, 0xbf, 0xf7, 0xbe, 0x15, 0x40, 0x7d, 0x23, 0x6f, 0x5e, 0x56, 0x35, 0x97, 0x9c,
	0x74, 0xf0, 0xdb, 0x1f, 0x02, 0xac, 0xa6, 0xcb, 0x45, 0x25, 0x73, 0x5e, 0x0a, 0xff, 0xe7, 0x36,
	0x0c, 0x8e, 0x79, 0xd0, 0xc4, 0x84, 0x40, 0x27, 0xe5, 0x19, 0xf3, 0x5a, 0xe3, 0xd6, 0x51, 0x97,
	0xaa, 0x6f, 0xc4, 0xe4, 0x6d, 0xc5, 0xbc, 0xb6, 0xc6, 0xf0, 0x9b, 0x7c, 0x06, 0xa3, 0x35, 0x4f,
	0x13, 0x4c, 0x8a, 0xab, 0x44, 0x5e, 0x79, 0xd6, 0xd8, 0x3a, 0x72, 0xe8, 0xb0, 0x01, 0x97, 0x89,
	0xbc, 0xc2, 0x44, 0xc5, 0x75, 0xc6, 0xad, 0x23, 0x87, 0xaa, 0x6f, 0xf2, 0x0c, 0xf6, 0x52, 0x5e,
	0x4a, 0x56, 0xca, 0xf8, 0x82, 0xd7, 0x45, 0x22, 0xbd, 0xae, 0x2a, 0x3b, 0x32, 0xe8, 0x37, 0x0a,
	0x24, 0x1f, 0x83, 0xb3, 0xa9, 0xf3, 0xf8, 0xed, 0x86, 0xd5, 0xb7, 0x9e, 0xad, 0x6a, 0xf7, 0x37,
	0x75, 0xfe, 0x06, 0x63, 0xf2, 0x21, 0xd8, 0x49, 0x9a, 0xb2, 0x4a, 0x7a, 0x3d, 0x95, 0x6b, 0x22,
	0xf2, 0x10, 0xba, 0x92, 0x5f, 0xb3, 0xd2, 0x73, 0xc6, 0xad, 0x23, 0x8b, 0xea, 0x80, 0x3c, 0x87,
	0x7d, 0x99, 0x17, 0x8c, 0x6f, 0x64, 0x2c, 0x58, 0xca, 0xcb, 0x4c, 0x78, 0xa0, 0xd2, 0xf6, 0x0c,
	0x7c, 0xae, 0x51, 0xff, 0xb7, 0x36, 0xf4, 0xce, 0x98, 0x10, 0xc9, 0x25, 0x23, 0x7b, 0xd0, 0xce,
	0x33, 0xa5, 0x82, 0x45, 0xdb, 0x79, 0x46, 0x9e, 0xed, 0x68, 0xb0, 0x37, 0x39, 0x78, 0xa9, 0x64,
	0x35, 0x8b, 0xa3, 0xdb, 0x8a, 0x19, 0x59, 0x3e, 0x01, 0x07, 0x8b, 0x0a, 0x99, 0x14, 0x95, 0x67,
	0xa9, 0xec, 0x3b, 0x00, 0xef, 0x5e, 0xb3, 0x82, 0x4b, 0x16, 0x27, 0x59, 0x56, 0x33, 0x21, 0x94,
	0x32, 0x43, 0x3a, 0xd2, 0x68, 0xa0, 0x41, 0xf2, 0x04, 0x06, 0x66, 0x59, 0xc5, 0xeb, 0x46, 0x1f,
	0xd0, 0xd0, 0x92, 0xd7, 0x92, 0x3c, 0x06, 0x40, 0x9d, 0xd7, 0x9a, 0xb7, 0x15, 0xef, 0x28, 0x44,
	0xd1, 0x1e, 0xf4, 0xaa, 0xe4, 0x76, 0xcd, 0x93, 0x4c, 0xe9, 0x33, 0xa4, 0x4d, 0x88, 0xb7, 0x48,
	0x79, 0x52, 0x79, 0xfd, 0x71, 0xeb, 0x68, 0xd0, 0xdc, 0x62, 0xa7, 0xfd, 0x54, 0xd1, 0xc4, 0x07,
	0x6b, 0x93, 0x55, 0x4a, 0xc5, 0xc1, 0xc4, 0xd5, 0xab, 0xee, 0x66, 0x86, 0x22, 0xe9, 0x7f, 0x05,
	0xf6, 0xa2, 0xce, 0x2f, 0xf3, 0x92, 0x3c, 0x02, 0x3b, 0xa9, 0xca, 0xd8, 0xc8, 0xd5, 0xa5, 0xdd,
	0xa4, 0x2a, 0x67, 0x19, 0xc2, 0x65, 0x22, 0x10, 0x6e, 0x8f, 0x2d, 0x84, 0xcb, 0x44, 0xcc, 0x32,
	0xff, 0xd7, 0x16, 0xec, 0xaf, 0x2a, 0x21, 0x6b, 0x96, 0x14, 0x94, 0xbd, 0xdd, 0x30, 0x21, 0xc9,
	0x53, 0xb0, 0xb9, 0xaa, 0xa5, 0x2a, 0x0c, 0x26, 0x43, 0xbd, 0xa5, 0xae, 0x4f, 0x0d, 0x47, 0x3e,
	0x05, 0xa8, 0x59, 0xc6, 0xd6, 0xf9, 0x3b, 0x9c, 0x09, 0x6c, 0x44, 0x9f, 0xee, 0x20, 0xe4, 0x09,
	0x58, 0x85, 0xb8, 0x54, 0x72, 0x0d, 0x26, 0xa3, 0x7b, 0x1d, 0xa2, 0xc8, 0x90, 0xcf, 0xe1, 0x80,
	0xdd, 0x54, 0x2c, 0x95, 0x71, 0xc6, 0xdf, 0x97, 0xfa, 0x08, 0x4a, 0xbd, 0x3e, 0x75, 0x35, 0x31,
	0xdd, 0xe2, 0xfe, 0x97, 0x40, 0xee, 0x22, 0xca, 0x44, 0xc5, 0x4b, 0xc1, 0x9a, 0x3d, 0xda, 0xff,
	0xb4, 0x87, 0xff, 0x03, 0x1c, 0xec, 0xa6, 0xfd, 0x9f, 0xfb, 0xfd, 0xb7, 0x11, 0xf3, 0x05, 0x40,
	0x90, 0x5e, 0x37, 0xa5, 0x1f, 0x03, 0x14, 0x7a, 0x49, 0xbc, 0x9d, 0x57, 0xc7, 0x20, 0xb3, 0x8c,
	0x3c, 0x07, 0x9b, 0x32, 0xb1, 0x59, 0x4b, 0x53, 0x75, 0x5f, 0x57, 0x0d, 0xeb, 0x9a, 0xd7, 0xc7,
	0x3c, 0x63, 0xd4, 0xd0, 0x58, 0x07, 0x5b, 0x1f, 0x6b, 0xff, 0x98, 0xc9, 0x45, 0x24, 0x42, 0xc0,
	0x1f, 0xc1, 0x40, 0x6d, 0xaa, 0x65, 0xf0, 0x7f, 0x6f, 0xc3, 0x28, 0x48, 0x53, 0x26, 0x44, 0x73,
	0x0e, 0x02, 0x9d, 0xbc, 0x10, 0xb9, 0x39, 0x81, 0xfa, 0xc6, 0x71, 0xd7, 0x13, 0xc0, 0x4a, 0x99,
	0x5f, 0xe4, 0xac, 0x56, 0x87, 0x70, 0xe8, 0x48, 0x4d, 0x42, 0x03, 0x92, 0x43, 0xe8, 0x6f, 0x04,
	0xab, 0xcb, 0xa4, 0x60, 0x6a, 0x63, 0x74, 0xba, 0x89, 0x91, 0xab, 0x12, 0x21, 0xde, 0xf3, 0x3a,
	0x33, 0x5e, 0xd9, 0xc6, 0xe4, 0x0b, 0x20, 0xb8, 0x2e, 0xde, 0xbe, 0x43, 0x79, 0x79, 0xc1, 0x55,
	0xfb, 0x87, 0xd4, 0x45, 0xe6, 0xd4, 0x10, 0xb3, 0xf2, 0x82, 0x93, 0x31, 0x0c, 0xf1, 0x50, 0x71,
	0x91, 0xa6, 0x71, 0x51, 0xa6, 0xaa, 0xef, 0x0e, 0x05, 0xc4, 0xce, 0xd2, 0xf4, 0xac, 0x4c, 0xd1,
	0x76, 0x85, 0x88, 0xd1, 0xad, 0x3f, 0xf1, 0x92, 0x19, 0xeb, 0x40, 0x21, 0x22, 0x83, 0xe0, 0xb3,
	0x93, 0x17, 0x2c, 0x17, 0xef, 0x94, 0x7f, 0x1c, 0x6a, 0x22, 0xf2, 0xd4, 0xdc, 0xb3, 0xda, 0xda,
	0xda, 0x51, 0xb9, 0x43, 0xbc, 0x67, 0x65, 0x5c, 0xed, 0x33, 0xd8, 0x6b, 0x24, 0x33, 0xc3, 0x74,
	0x08, 0x7d, 0xfd, 0x70, 0x31, 0xdd, 0xb9, 0x3e, 0xdd, 0xc6, 0xd8, 0x8f, 0x9d, 0x7a, 0x6d, 0x55,
	0xcf, 0xc9, 0x9b, 0x62, 0x68, 0x71, 0xd3, 0x64, 0x23, 0x59, 0x13, 0xbe, 0x78, 0x03, 0x83, 0x9d,
	0x99, 0x21, 0x3d, 0xb0, 0x56, 0xd3, 0xa5, 0xfb, 0x80, 0xb8, 0x30, 0x44, 0xa3, 0x37, 0xd6, 0x73,
	0x5b, 0x64, 0x08, 0x7d, 0x44, 0x96, 0x9b, 0xf5, 0xda, 0x6d, 0xdf, 0x45, 0xe2, 0xca, 0xb5, 0xc8,
	0x00, 0x7a, 0xab, 0xa9, 0xa6, 0x3a, 0x2f, 0x7e, 0x69, 0x81, 0xb3, 0x9d, 0x18, 0xa4, 0xce, 0x57,
	0xc7, 0xc7, 0xe1, 0xf9, 0xb9, 0xfb, 0x80, 0x8c, 0xc0, 0x89, 0x16, 0x8b, 0xf8, 0x34, 0xa0, 0x27,
	0xa1, 0xdb, 0x42, 0x6e, 0x1e, 0x46, 0xdf, 0x2f, 0xe8, 0x77, 0x6e, 0x9b, 0xec, 0xc3, 0x60, 0xbe,
	0x88, 0xe2, 0x6f, 0x83, 0xf9, 0xf4, 0x34, 0x9c, 0xba, 0x96, 0x3a, 0xc2, 0xe9, 0x2c, 0x9c, 0x47,
	0x71, 0x48, 0xe9, 0x82, 0xba, 0x1d, 0x4c, 0x5f, 0x06, 0x34, 0x38, 0x0b, 0xa3, 0x90, 0xba, 0x5d,
	0x3c, 0xc3, 0x6c, 0x1e, 0x85, 0x74, 0x1e, 0x9c, 0xba, 0x36, 0x16, 0x5b, 0x86, 0xf3, 0xe9, 0x6c,
	0x7e, 0xe2, 0xf6, 0x30, 0x88, 0x66, 0x67, 0xe1, 0x62, 0x15, 0xb9, 0xfd, 0xc9, 0x1f, 0x2d, 0xe8,
	0xd0, 0x1b, 0x79, 0x43, 0x5e, 0x03, 0x2c, 0x37, 0xb2, 0x79, 0xb3, 0x1f, 0x99, 0x97, 0xea, 0xfe,
	0xeb, 0x72, 0xe8, 0x69, 0xf8, 0x6f, 0xdc, 0xfc, 0x35, 0xc0, 0x09, 0xdb, 0xa6, 0x7f, 0xf4, 0xd7,
	0x75, 0xff, 0x56, 0xe0, 0x05, 0x58, 0x41, 0x7a, 0x4d, 0xcc, 0x13, 0x79, 0x67, 0xcb, 0xc3, 0x83,
	0x1d, 0x44, 0xaf, 0x9d, 0xbc, 0x06, 0x9b, 0x06, 0xd3, 0xd9, 0xea, 0x9c, 0xbc, 0x02, 0x5b, 0x4f,
	0x02, 0xf9, 0xa0, 0x59, 0xb6, 0x63, 0xa5, 0xc3, 0x87, 0xf7, 0x41, 0x9d, 0xfe, 0xa3, 0xad, 0xfe,
	0xe1, 0xaf, 0xfe, 0x0c, 0x00, 0x00, 0xff, 0xff, 0x65, 0x05, 0xb5, 0x57, 0xd1, 0x07, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// RxtxClient is the client API for Rxtx service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type RxtxClient interface {
	// PutMessage sends an upstream message. The service assumes responsibility
	// for the message when a response is sent.
	PutMessage(ctx context.Context, in *UpstreamRequest, opts ...grpc.CallOption) (*DownstreamResponse, error)
	// GetMessage returns an downstream/outbound (unsolicited) message to a
	// device.
	GetMessage(ctx context.Context, in *DownstreamRequest, opts ...grpc.CallOption) (*DownstreamResponse, error)
	// Ack acknowledges receipt and status of a message. If there's an error
	// handling the message the Result field in the request contains the error.
	Ack(ctx context.Context, in *AckRequest, opts ...grpc.CallOption) (*AckResponse, error)
}

type rxtxClient struct {
	cc *grpc.ClientConn
}

func NewRxtxClient(cc *grpc.ClientConn) RxtxClient {
	return &rxtxClient{cc}
}

func (c *rxtxClient) PutMessage(ctx context.Context, in *UpstreamRequest, opts ...grpc.CallOption) (*DownstreamResponse, error) {
	out := new(DownstreamResponse)
	err := c.cc.Invoke(ctx, "/rxtx.Rxtx/PutMessage", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rxtxClient) GetMessage(ctx context.Context, in *DownstreamRequest, opts ...grpc.CallOption) (*DownstreamResponse, error) {
	out := new(DownstreamResponse)
	err := c.cc.Invoke(ctx, "/rxtx.Rxtx/GetMessage", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rxtxClient) Ack(ctx context.Context, in *AckRequest, opts ...grpc.CallOption) (*AckResponse, error) {
	out := new(AckResponse)
	err := c.cc.Invoke(ctx, "/rxtx.Rxtx/Ack", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RxtxServer is the server API for Rxtx service.
type RxtxServer interface {
	// PutMessage sends an upstream message. The service assumes responsibility
	// for the message when a response is sent.
	PutMessage(context.Context, *UpstreamRequest) (*DownstreamResponse, error)
	// GetMessage returns an downstream/outbound (unsolicited) message to a
	// device.
	GetMessage(context.Context, *DownstreamRequest) (*DownstreamResponse, error)
	// Ack acknowledges receipt and status of a message. If there's an error
	// handling the message the Result field in the request contains the error.
	Ack(context.Context, *AckRequest) (*AckResponse, error)
}

func RegisterRxtxServer(s *grpc.Server, srv RxtxServer) {
	s.RegisterService(&_Rxtx_serviceDesc, srv)
}

func _Rxtx_PutMessage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpstreamRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RxtxServer).PutMessage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rxtx.Rxtx/PutMessage",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RxtxServer).PutMessage(ctx, req.(*UpstreamRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Rxtx_GetMessage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DownstreamRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RxtxServer).GetMessage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rxtx.Rxtx/GetMessage",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RxtxServer).GetMessage(ctx, req.(*DownstreamRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Rxtx_Ack_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AckRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RxtxServer).Ack(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rxtx.Rxtx/Ack",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RxtxServer).Ack(ctx, req.(*AckRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Rxtx_serviceDesc = grpc.ServiceDesc{
	ServiceName: "rxtx.Rxtx",
	HandlerType: (*RxtxServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PutMessage",
			Handler:    _Rxtx_PutMessage_Handler,
		},
		{
			MethodName: "GetMessage",
			Handler:    _Rxtx_GetMessage_Handler,
		},
		{
			MethodName: "Ack",
			Handler:    _Rxtx_Ack_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "rxtx.proto",
}

// RADIUSClient is the client API for RADIUS service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type RADIUSClient interface {
	Access(ctx context.Context, in *AccessRequest, opts ...grpc.CallOption) (*AccessResponse, error)
}

type rADIUSClient struct {
	cc *grpc.ClientConn
}

func NewRADIUSClient(cc *grpc.ClientConn) RADIUSClient {
	return &rADIUSClient{cc}
}

func (c *rADIUSClient) Access(ctx context.Context, in *AccessRequest, opts ...grpc.CallOption) (*AccessResponse, error) {
	out := new(AccessResponse)
	err := c.cc.Invoke(ctx, "/rxtx.RADIUS/Access", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RADIUSServer is the server API for RADIUS service.
type RADIUSServer interface {
	Access(context.Context, *AccessRequest) (*AccessResponse, error)
}

func RegisterRADIUSServer(s *grpc.Server, srv RADIUSServer) {
	s.RegisterService(&_RADIUS_serviceDesc, srv)
}

func _RADIUS_Access_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AccessRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RADIUSServer).Access(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rxtx.RADIUS/Access",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RADIUSServer).Access(ctx, req.(*AccessRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _RADIUS_serviceDesc = grpc.ServiceDesc{
	ServiceName: "rxtx.RADIUS",
	HandlerType: (*RADIUSServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Access",
			Handler:    _RADIUS_Access_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "rxtx.proto",
}
