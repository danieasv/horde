package apitoolbox

//
// Copyright 2020 Telenor Digital AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/eesrc/horde/pkg/output/outputconfig"

	"google.golang.org/grpc/codes"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/addons/magpie/datastore"
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/model"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc/status"
)

// NewCollectionFromModel returns an apipb.Collection instance from the
// model.Collection instance.
func NewCollectionFromModel(c model.Collection) *apipb.Collection {
	return &apipb.Collection{
		CollectionId: &wrappers.StringValue{Value: c.ID.String()},
		TeamId:       &wrappers.StringValue{Value: c.TeamID.String()},
		FieldMask:    NewFieldMaskFromModel(c.FieldMask),
		Firmware:     NewCollectionFirmwareConfigFromModel(c.Firmware),
		Tags:         c.Tags.TagMap,
	}
}

// NewFieldMaskFromModel converts a model.FieldMask into apipb.FieldMask
func NewFieldMaskFromModel(m model.FieldMask) *apipb.FieldMask {
	return &apipb.FieldMask{
		Imsi:     &wrappers.BoolValue{Value: m.IsSet(model.IMSIMask)},
		Imei:     &wrappers.BoolValue{Value: m.IsSet(model.IMEIMask)},
		Location: &wrappers.BoolValue{Value: m.IsSet(model.LocationMask)},
		Msisdn:   &wrappers.BoolValue{Value: m.IsSet(model.MSISDNMask)},
	}
}

// NewCollectionFirmwareConfigFromModel converts the model.CollectionFirmwareMetadata
// value into apipb.CollectionFirmware
func NewCollectionFirmwareConfigFromModel(m model.CollectionFirmwareMetadata) *apipb.CollectionFirmware {
	ret := &apipb.CollectionFirmware{
		Management: apipb.CollectionFirmware_disabled,
	}
	switch m.Management {
	case model.CollectionManagement:
		ret.Management = apipb.CollectionFirmware_collection
	case model.DeviceManagement:
		ret.Management = apipb.CollectionFirmware_device
	default:
		ret.Management = apipb.CollectionFirmware_disabled
	}
	if m.CurrentFirmwareID != 0 {
		ret.CurrentFirmwareId = &wrappers.StringValue{Value: m.CurrentFirmwareID.String()}
	}
	if m.TargetFirmwareID != 0 {
		ret.TargetFirmwareId = &wrappers.StringValue{Value: m.TargetFirmwareID.String()}
	}
	return ret
}

// NewDeviceFromModel converts model.Device into apipb.Device
func NewDeviceFromModel(d model.Device, c model.Collection) *apipb.Device {
	ret := &apipb.Device{
		DeviceId:     &wrappers.StringValue{Value: d.ID.String()},
		CollectionId: &wrappers.StringValue{Value: d.CollectionID.String()},
		Imsi:         &wrappers.StringValue{Value: fmt.Sprintf("%d", d.IMSI)},
		Imei:         &wrappers.StringValue{Value: fmt.Sprintf("%d", d.IMEI)},
		Tags:         d.Tags.TagMap,
		Network:      NewNetworkMetadataFromModel(d.Network, c.FieldMask),
		Firmware:     NewFirmwareMetadataFromModel(d.Firmware),
	}

	if c.FieldMask.IsSet(model.IMEIMask) {
		ret.Imei = nil
	}
	if c.FieldMask.IsSet(model.IMSIMask) {
		ret.Imsi = nil
	}
	// override with values from collection if collection management is used.
	if c.Firmware.Management == model.CollectionManagement {
		ret.Firmware.CurrentFirmwareId = nil
		if c.Firmware.CurrentFirmwareID != 0 {
			ret.Firmware.CurrentFirmwareId = &wrappers.StringValue{Value: c.Firmware.CurrentFirmwareID.String()}
		}
		ret.Firmware.TargetFirmwareId = nil
		if c.Firmware.TargetFirmwareID != 0 {
			ret.Firmware.TargetFirmwareId = &wrappers.StringValue{Value: c.Firmware.TargetFirmwareID.String()}
		}
	}
	return ret
}

// NewNetworkMetadataFromModel converts model.DeviceNetworkMetadata into apipb.NetworkMetadata
func NewNetworkMetadataFromModel(m model.DeviceNetworkMetadata, fieldMask model.FieldMask) *apipb.NetworkMetadata {
	allocTime := timeToMillis(m.AllocatedAt)
	if allocTime < 0 {
		allocTime = 0
	}
	ret := &apipb.NetworkMetadata{
		AllocatedIp: &wrappers.StringValue{Value: m.AllocatedIP},
		AllocatedAt: &wrappers.DoubleValue{Value: allocTime},
		CellId:      &wrappers.Int64Value{Value: m.CellID},
	}
	if fieldMask.IsSet(model.LocationMask) {
		ret.CellId = nil
	}
	return ret
}

// NewFirmwareFromModel converts a model.Firmware entity into the corresponding apipb.Firmware type
func NewFirmwareFromModel(fw model.Firmware) *apipb.Firmware {
	return &apipb.Firmware{
		ImageId:      &wrappers.StringValue{Value: fw.ID.String()},
		Version:      &wrappers.StringValue{Value: fw.Version},
		Filename:     &wrappers.StringValue{Value: fw.Filename},
		Sha256:       &wrappers.StringValue{Value: fw.SHA256},
		Length:       &wrappers.Int32Value{Value: int32(fw.Length)},
		CollectionId: &wrappers.StringValue{Value: fw.CollectionID.String()},
		Tags:         fw.TagData(),
	}
}

// NewFirmwareMetadataFromModel converts model.DeviceFirmwareMetadata into apipb.FirmwareMetadata
func NewFirmwareMetadataFromModel(m model.DeviceFirmwareMetadata) *apipb.FirmwareMetadata {
	state := apipb.FirmwareMetadata_Current
	switch m.State {
	case model.Current:
		state = apipb.FirmwareMetadata_Current
	case model.Initializing:
		state = apipb.FirmwareMetadata_Initializing
	case model.Pending:
		state = apipb.FirmwareMetadata_Pending
	case model.Downloading:
		state = apipb.FirmwareMetadata_Downloading
	case model.Completed:
		state = apipb.FirmwareMetadata_Completed
	case model.UpdateFailed:
		state = apipb.FirmwareMetadata_UpdateFailed
	case model.TimedOut:
		state = apipb.FirmwareMetadata_TimedOut
	case model.Reverted:
		state = apipb.FirmwareMetadata_Reverted
	default:
		// Unknown state - set to current
		state = apipb.FirmwareMetadata_Current
	}
	return &apipb.FirmwareMetadata{
		CurrentFirmwareId: &wrappers.StringValue{Value: m.CurrentFirmwareID.String()},
		TargetFirmwareId:  &wrappers.StringValue{Value: m.TargetFirmwareID.String()},
		SerialNumber:      &wrappers.StringValue{Value: m.SerialNumber},
		ModelNumber:       &wrappers.StringValue{Value: m.ModelNumber},
		Manufacturer:      &wrappers.StringValue{Value: m.Manufacturer},
		FirmwareVersion:   &wrappers.StringValue{Value: m.FirmwareVersion},
		StateMessage:      &wrappers.StringValue{Value: m.StateMessage},
		State:             &wrappers.StringValue{Value: state.String()},
	}
}

// NewOutputFromModel converts model.Output into apipb.Output
func NewOutputFromModel(o model.Output) *apipb.Output {
	ret := &apipb.Output{
		OutputId:     &wrappers.StringValue{Value: o.ID.String()},
		CollectionId: &wrappers.StringValue{Value: o.CollectionID.String()},
		Enabled:      &wrappers.BoolValue{Value: o.Enabled},
		Config:       &apipb.OutputConfig{},
		Tags:         o.TagMap,
	}
	switch o.Type {
	case "udp":
		ret.Type = apipb.Output_udp
		tmp, ok := o.Config[outputconfig.UDPHost]
		if ok {
			ret.Config.Host = &wrappers.StringValue{Value: tmp.(string)}
		}
		tmp, ok = o.Config[outputconfig.UDPPort]
		if ok {
			ret.Config.Port = &wrappers.Int32Value{Value: int32(tmp.(float64))}
		}

	case "mqtt":
		ret.Type = apipb.Output_mqtt
		tmp, ok := o.Config[outputconfig.MQTTEndpoint]
		if ok {
			ret.Config.Endpoint = &wrappers.StringValue{Value: tmp.(string)}
		}
		tmp, ok = o.Config[outputconfig.MQTTUsername]
		if ok {
			ret.Config.Username = &wrappers.StringValue{Value: tmp.(string)}
		}
		tmp, ok = o.Config[outputconfig.MQTTPassword]
		if ok {
			ret.Config.Password = &wrappers.StringValue{Value: tmp.(string)}
		}
		tmp, ok = o.Config[outputconfig.MQTTClientID]
		if ok {
			ret.Config.ClientId = &wrappers.StringValue{Value: tmp.(string)}
		}
		tmp, ok = o.Config[outputconfig.MQTTTopicName]
		if ok {
			ret.Config.TopicName = &wrappers.StringValue{Value: tmp.(string)}
		}
		tmp, ok = o.Config[outputconfig.MQTTDisableCertCheck]
		if ok {
			ret.Config.DisableCertCheck = &wrappers.BoolValue{Value: tmp.(bool)}
		}

	case "ifttt":
		ret.Type = apipb.Output_ifttt
		tmp, ok := o.Config[outputconfig.IFTTTKey]
		if ok {
			ret.Config.Key = &wrappers.StringValue{Value: tmp.(string)}
		}
		tmp, ok = o.Config[outputconfig.IFTTTEvent]
		if ok {
			ret.Config.EventName = &wrappers.StringValue{Value: tmp.(string)}
		}
		tmp, ok = o.Config[outputconfig.FTTTAsIsPayload]
		if ok {
			ret.Config.AsIsPayload = &wrappers.BoolValue{Value: tmp.(bool)}
		}

	default: // really "webhook":
		ret.Type = apipb.Output_webhook
		tmp, ok := o.Config[outputconfig.WebhookURLField]
		if ok {
			ret.Config.Url = &wrappers.StringValue{Value: tmp.(string)}
		}
		tmp, ok = o.Config[outputconfig.WebhookBasicAuthUser]
		if ok {
			ret.Config.BasicAuthUser = &wrappers.StringValue{Value: tmp.(string)}
		}
		tmp, ok = o.Config[outputconfig.WebhookBasicAuthPass]
		if ok {
			ret.Config.BasicAuthPass = &wrappers.StringValue{Value: tmp.(string)}
		}
		tmp, ok = o.Config[outputconfig.WebhookCustomHeaderName]
		if ok {
			ret.Config.CustomHeaderName = &wrappers.StringValue{Value: tmp.(string)}
		}
		tmp, ok = o.Config[outputconfig.WebhookCustomHeaderValue]
		if ok {
			ret.Config.CustomHeaderValue = &wrappers.StringValue{Value: tmp.(string)}
		}
	}
	return ret
}

// NewOutputLogsFromModel returns the log entries converted to apipb types
func NewOutputLogsFromModel(logs []model.OutputLogEntry) []*apipb.OutputLogEntry {
	if logs == nil {
		return nil
	}
	ret := make([]*apipb.OutputLogEntry, 0)
	for _, v := range logs {
		ret = append(ret, &apipb.OutputLogEntry{
			Time:     &wrappers.DoubleValue{Value: timeToMillis(v.Time)},
			Message:  &wrappers.StringValue{Value: v.Message},
			Repeated: &wrappers.Int32Value{Value: int32(v.Repeated)},
		})
	}
	return ret
}

// NewOutputStatusFromModel converts an internal status structure into the apipb
// equivalent.
func NewOutputStatusFromModel(collectionID string, outputID string, enabled bool, status model.OutputStatus) *apipb.OutputStatus {
	return &apipb.OutputStatus{
		CollectionId: &wrappers.StringValue{Value: collectionID},
		OutputId:     &wrappers.StringValue{Value: outputID},
		ErrorCount:   &wrappers.Int32Value{Value: int32(status.ErrorCount)},
		Forwarded:    &wrappers.Int32Value{Value: int32(status.Forwarded)},
		Received:     &wrappers.Int32Value{Value: int32(status.Received)},
		Retransmits:  &wrappers.Int32Value{Value: int32(status.Retransmits)},
		Enabled:      &wrappers.BoolValue{Value: enabled},
	}
}

// NewOutputConfigFromAPI generates a model config based on protobuffer request object
func NewOutputConfigFromAPI(o *apipb.Output) model.OutputConfig {
	ret := make(model.OutputConfig)
	if o.Config == nil || o.Type == apipb.Output_undefined {
		return ret
	}
	if o.Config.AsIsPayload != nil {
		ret[outputconfig.FTTTAsIsPayload] = o.Config.AsIsPayload.Value
	}
	if o.Config.EventName != nil {
		ret[outputconfig.IFTTTEvent] = o.Config.EventName.Value
	}
	if o.Config.Key != nil {
		ret[outputconfig.IFTTTKey] = o.Config.Key.Value
	}
	if o.Config.Url != nil {
		ret[outputconfig.WebhookURLField] = o.Config.Url.Value
	}
	if o.Config.BasicAuthPass != nil {
		ret[outputconfig.WebhookBasicAuthPass] = o.Config.BasicAuthPass.Value
	}
	if o.Config.BasicAuthUser != nil {
		ret[outputconfig.WebhookBasicAuthUser] = o.Config.BasicAuthUser.Value
	}
	if o.Config.CustomHeaderName != nil {
		ret[outputconfig.WebhookCustomHeaderName] = o.Config.CustomHeaderName.Value
	}
	if o.Config.CustomHeaderValue != nil {
		ret[outputconfig.WebhookCustomHeaderValue] = o.Config.CustomHeaderValue.Value
	}
	if o.Config.ClientId != nil {
		ret[outputconfig.MQTTClientID] = o.Config.ClientId.Value
	}
	if o.Config.DisableCertCheck != nil {
		ret[outputconfig.MQTTDisableCertCheck] = o.Config.DisableCertCheck.Value
	}
	if o.Config.Endpoint != nil {
		ret[outputconfig.MQTTEndpoint] = o.Config.Endpoint.Value
	}
	if o.Config.Password != nil {
		ret[outputconfig.MQTTPassword] = o.Config.Password.Value
	}
	if o.Config.TopicName != nil {
		ret[outputconfig.MQTTTopicName] = o.Config.TopicName.Value
	}
	if o.Config.Username != nil {
		ret[outputconfig.MQTTUsername] = o.Config.Username.Value
	}
	if o.Config.Host != nil {
		ret[outputconfig.UDPHost] = o.Config.Host.Value
	}
	if o.Config.Port != nil {
		ret[outputconfig.UDPPort] = float64(o.Config.Port.Value)
	}
	return ret
}

// NewTokenFromModel converts a model.Token into apipb.Token
func NewTokenFromModel(token model.Token) *apipb.Token {
	return &apipb.Token{
		Resource: &wrappers.StringValue{Value: token.Resource},
		Write:    &wrappers.BoolValue{Value: token.Write},
		Token:    &wrappers.StringValue{Value: token.Token},
		Tags:     token.TagData(),
	}
}

// NewTeamFromModel converts a model.Team into apipb.Team
func NewTeamFromModel(team model.Team, includeMembers bool) *apipb.Team {
	ret := &apipb.Team{
		TeamId: &wrappers.StringValue{Value: team.ID.String()},
		Tags:   team.TagData(),
	}
	if includeMembers {
		ret.Members = make([]*apipb.Member, 0)
		for _, v := range team.Members {
			ret.Members = append(ret.Members, NewMemberFromModel(team.ID, v))
		}
	}
	return ret
}

// NewMemberFromModel creates an apipb.Member instace from a model.Member instance
func NewMemberFromModel(teamID model.TeamKey, member model.Member) *apipb.Member {
	ret := &apipb.Member{
		UserId:        &wrappers.StringValue{Value: member.User.ID.String()},
		TeamId:        &wrappers.StringValue{Value: teamID.String()},
		Role:          &wrappers.StringValue{Value: member.Role.String()},
		Name:          &wrappers.StringValue{Value: member.User.Name},
		Email:         &wrappers.StringValue{Value: member.User.Email},
		VerifiedEmail: &wrappers.BoolValue{Value: member.User.VerifiedEmail},
		Phone:         &wrappers.StringValue{Value: member.User.Phone},
		VerifiedPhone: &wrappers.BoolValue{Value: member.User.VerifiedPhone},
		AvatarUrl:     &wrappers.StringValue{Value: member.User.AvatarURL},
	}

	if member.User.AuthType == model.AuthConnectID {
		ret.AuthType = &wrappers.StringValue{Value: "connect"}
		ret.ConnectId = &wrappers.StringValue{Value: member.User.ExternalID}
		ret.GitHubLogin = nil
	} else {
		ret.AuthType = &wrappers.StringValue{Value: "github"}
		ret.ConnectId = nil
		ret.GitHubLogin = &wrappers.StringValue{Value: member.User.ExternalID}
	}
	return ret
}

// TimeToMillis converts a time value into milliseconds
func timeToMillis(t time.Time) float64 {
	return math.Floor(float64(t.UnixNano()) / float64(time.Millisecond))
}

func nanosToMillis(nanos int64) float64 {
	return math.Floor(float64(nanos) / float64(time.Millisecond))
}

func milliToNano(ms int64) int64 {
	return ms * int64(time.Millisecond)
}

const collectionDataItemLimit = 256

// ApplyDataFilter creates a data filter instance from a apipb.ListMessagesRequest
// request object.
func ApplyDataFilter(req *apipb.ListMessagesRequest, dataFilter *datastore.DataFilter) {
	dataFilter.Limit = collectionDataItemLimit
	if req.Since != nil {
		dataFilter.From = milliToNano(req.Since.Value)
	}
	if req.Until != nil {
		dataFilter.To = milliToNano(req.Until.Value)
	}
	if req.Limit != nil {
		dataFilter.Limit = req.Limit.Value
	}
}

// NewInviteFromModel converts a model.Invite into an apipb.Invite type
func NewInviteFromModel(invite model.Invite) *apipb.Invite {
	return &apipb.Invite{
		Code:      &wrappers.StringValue{Value: invite.Code},
		CreatedAt: &wrappers.DoubleValue{Value: timeToMillis(invite.Created)},
	}
}

// NewUserProfileFromUser converts a model.User object into a UserProfile message
func NewUserProfileFromUser(user model.User) *apipb.UserProfile {
	ret := &apipb.UserProfile{
		Name:          &wrappers.StringValue{Value: user.Name},
		Email:         &wrappers.StringValue{Value: user.Email},
		Phone:         &wrappers.StringValue{Value: user.Phone},
		VerifiedEmail: &wrappers.BoolValue{Value: user.VerifiedEmail},
		VerifiedPhone: &wrappers.BoolValue{Value: user.VerifiedPhone},
		AvatarUrl:     &wrappers.StringValue{Value: user.AvatarURL},
	}

	switch user.AuthType {
	case model.AuthConnectID:
		ret.ConnectId = &wrappers.StringValue{Value: user.ExternalID}
		ret.ProfileUrl = &wrappers.StringValue{Value: "https://connect.telenordigital.com/gui/mypage/overview"}
		ret.Provider = &wrappers.StringValue{Value: "connect"}
		ret.LogoutUrl = &wrappers.StringValue{Value: "/connect/logout"}

	case model.AuthGitHub:
		ret.GithubLogin = &wrappers.StringValue{Value: user.ExternalID}
		ret.ProfileUrl = &wrappers.StringValue{Value: "https://github.com/settings/profile"}
		ret.Provider = &wrappers.StringValue{Value: "github"}
		ret.LogoutUrl = &wrappers.StringValue{Value: "/github/logout"}

	case model.AuthInternal:
		ret.Provider = &wrappers.StringValue{Value: "internal"}

	default:
		logging.Error("Unknown auth provider for user %d: %v", user.ID, user.AuthType)
	}
	return ret
}

// NewDownstreamMessage converts a SendMessageRequest into
// model.DownstreamMessage. If there is an error converting the message an error
// is returned. The error contains the appropriate status code.
func NewDownstreamMessage(msg *apipb.SendMessageRequest) (model.DownstreamMessage, error) {
	ret := model.DownstreamMessage{}

	if msg == nil {
		return ret, status.Error(codes.InvalidArgument, "Missing message")
	}

	// The UDP transport is the default
	ret.Transport = model.UDPTransport
	if msg.Transport != nil {
		ret.Transport = model.MessageTransportFromString(msg.Transport.Value)
	}
	if ret.Transport == model.UnknownTransport {
		return ret, status.Error(codes.InvalidArgument, "Unknown transport")
	}

	if ret.Transport == model.CoAPTransport {
		// Set the default CoAP port for push messages
		ret.Port = 5683
	}
	if msg.Port != nil {
		ret.Port = int(msg.Port.Value)
	}
	pull := ret.Transport == model.CoAPPullTransport || ret.Transport == model.UDPPullTransport
	// Port is required
	if !pull && ret.Port == 0 {
		return ret, status.Error(codes.InvalidArgument, "Port is required")
	}
	// Payload is - of course - required
	if msg.Payload == nil || len(msg.Payload) == 0 {
		return ret, status.Error(codes.InvalidArgument, "Empty payload")
	}
	ret.Payload = msg.Payload

	if msg.CoapPath != nil {
		ret.Path = msg.CoapPath.Value
	}

	// CoAP path is required for push messages
	if ret.Transport == model.CoAPTransport && strings.TrimSpace(ret.Path) == "" {
		return ret, status.Error(codes.InvalidArgument, "CoAP message needs a path")
	}

	return ret, nil
}

// NewOutputDataMessageFromModel converts a model.DataMessage into the
// apipb.OutputDataMessage equivalent
func NewOutputDataMessageFromModel(msg model.DataMessage, collection model.Collection) *apipb.OutputDataMessage {
	var coapMetadata *apipb.CoAPMetadata
	var udpMetadata *apipb.UDPMetadata
	transportType := "unknown"
	switch msg.Transport {
	case model.CoAPPullTransport:
		fallthrough
	case model.CoAPTransport:
		transportType = "coap"
		coapMetadata = &apipb.CoAPMetadata{
			Code: &wrappers.StringValue{Value: msg.CoAP.Code},
			Path: &wrappers.StringValue{Value: msg.CoAP.Path},
		}
	case model.UDPPullTransport:
		fallthrough
	case model.UDPTransport:
		transportType = "udp"
		udpMetadata = &apipb.UDPMetadata{
			LocalPort:  &wrappers.Int32Value{Value: int32(msg.UDP.LocalPort)},
			RemotePort: &wrappers.Int32Value{Value: int32(msg.UDP.RemotePort)},
		}
	default:
		logging.Warning("Unknown transport type (%s) for data message", msg.Transport.String())
	}
	// TODO(stalehd): Upgrade client to support int64/string formats (or deprecate the client)
	return &apipb.OutputDataMessage{
		Type:         apipb.OutputDataMessage_data,
		Device:       NewDeviceFromModel(msg.Device, collection),
		Payload:      msg.Payload,
		Received:     &wrappers.DoubleValue{Value: timeToMillis(msg.Received)},
		Transport:    transportType,
		CoapMetaData: coapMetadata,
		UdpMetaData:  udpMetadata,
	}
}
