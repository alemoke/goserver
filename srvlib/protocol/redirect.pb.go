// Code generated by protoc-gen-go.
// source: protocol/redirect.proto
// DO NOT EDIT!

package protocol

import proto "code.google.com/p/goprotobuf/proto"
import json "encoding/json"
import math "math"

// Reference proto, json, and math imports to suppress error if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type SSPacketRedirect struct {
	PacketId         *SrvlibPacketID `protobuf:"varint,1,req,enum=protocol.SrvlibPacketID,def=-2004" json:"PacketId,omitempty"`
	ClientSid        *int64          `protobuf:"varint,2,opt" json:"ClientSid,omitempty"`
	SrvRoutes        []*SrvInfo      `protobuf:"bytes,3,rep" json:"SrvRoutes,omitempty"`
	Data             []byte          `protobuf:"bytes,4,req" json:"Data,omitempty"`
	XXX_unrecognized []byte          `json:"-"`
}

func (m *SSPacketRedirect) Reset()         { *m = SSPacketRedirect{} }
func (m *SSPacketRedirect) String() string { return proto.CompactTextString(m) }
func (*SSPacketRedirect) ProtoMessage()    {}

const Default_SSPacketRedirect_PacketId SrvlibPacketID = SrvlibPacketID_PACKET_SS_REDIRECT

func (m *SSPacketRedirect) GetPacketId() SrvlibPacketID {
	if m != nil && m.PacketId != nil {
		return *m.PacketId
	}
	return Default_SSPacketRedirect_PacketId
}

func (m *SSPacketRedirect) GetClientSid() int64 {
	if m != nil && m.ClientSid != nil {
		return *m.ClientSid
	}
	return 0
}

func (m *SSPacketRedirect) GetSrvRoutes() []*SrvInfo {
	if m != nil {
		return m.SrvRoutes
	}
	return nil
}

func (m *SSPacketRedirect) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

type SrvInfo struct {
	SArea            *int32 `protobuf:"varint,1,req" json:"SArea,omitempty"`
	SType            *int32 `protobuf:"varint,2,req" json:"SType,omitempty"`
	SId              *int32 `protobuf:"varint,3,req" json:"SId,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *SrvInfo) Reset()         { *m = SrvInfo{} }
func (m *SrvInfo) String() string { return proto.CompactTextString(m) }
func (*SrvInfo) ProtoMessage()    {}

func (m *SrvInfo) GetSArea() int32 {
	if m != nil && m.SArea != nil {
		return *m.SArea
	}
	return 0
}

func (m *SrvInfo) GetSType() int32 {
	if m != nil && m.SType != nil {
		return *m.SType
	}
	return 0
}

func (m *SrvInfo) GetSId() int32 {
	if m != nil && m.SId != nil {
		return *m.SId
	}
	return 0
}

func init() {
}
