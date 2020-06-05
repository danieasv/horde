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
	"testing"
	"time"

	"github.com/eesrc/horde/pkg/addons/magpie/datastore"
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/output/outputconfig"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/require"
)

func TestCollectionConversion(t *testing.T) {
	assert := require.New(t)

	c := model.NewCollection()
	c.SetTag("name", "Some name")
	c.SetTag("tag", "Some tag")
	c.ID = model.CollectionKey(0)
	c.TeamID = model.TeamKey(1)
	c.Firmware.CurrentFirmwareID = model.FirmwareKey(2)
	c.Firmware.TargetFirmwareID = model.FirmwareKey(3)
	c.Firmware.Management = model.CollectionManagement
	c.FieldMask = model.FieldMask(model.IMSIMask | model.IMEIMask)
	n := NewCollectionFromModel(c)

	assert.Equal("Some name", n.Tags["name"])
	assert.Equal("Some tag", n.Tags["tag"])
	assert.Equal(c.ID.String(), n.CollectionId.Value)
	assert.Equal(c.TeamID.String(), n.TeamId.Value)
	assert.True(n.FieldMask.Imei.Value)
	assert.True(n.FieldMask.Imsi.Value)
	assert.False(n.FieldMask.Location.Value)
	assert.Equal(c.Firmware.CurrentFirmwareID.String(), n.Firmware.CurrentFirmwareId.Value)
	assert.Equal(c.Firmware.TargetFirmwareID.String(), n.Firmware.TargetFirmwareId.Value)
	assert.Equal(apipb.CollectionFirmware_collection, n.Firmware.Management)

	c.Firmware.Management = model.DeviceManagement
	n = NewCollectionFromModel(c)
	assert.Equal(apipb.CollectionFirmware_device, n.Firmware.Management)

	c.Firmware.Management = model.DisabledManagement
	n = NewCollectionFromModel(c)
	assert.Equal(apipb.CollectionFirmware_disabled, n.Firmware.Management)
}

func TestDeviceConversion(t *testing.T) {
	assert := require.New(t)

	d := model.NewDevice()
	d.SetTag("name", "Some name")
	d.ID = model.DeviceKey(1)
	d.CollectionID = model.CollectionKey(2)
	d.IMEI = 1
	d.IMSI = 2
	d.Firmware = model.DeviceFirmwareMetadata{
		CurrentFirmwareID: model.FirmwareKey(1),
		TargetFirmwareID:  model.FirmwareKey(2),
		SerialNumber:      "1",
		ModelNumber:       "2",
		Manufacturer:      "3",
		FirmwareVersion:   "4",
		State:             model.Downloading,
		StateMessage:      "text message",
	}
	d.Network = model.DeviceNetworkMetadata{
		CellID:      1,
		AllocatedAt: time.Now(),
		AllocatedIP: "1.2.3.4",
		ApnID:       1,
		NasID:       2,
	}

	c := model.NewCollection()
	c.ID = model.CollectionKey(2)
	c.Firmware.Management = model.DeviceManagement
	c.FieldMask = 0
	n := NewDeviceFromModel(d, c)

	assert.Equal(d.ID.String(), n.DeviceId.Value)
	assert.Equal(d.CollectionID.String(), n.CollectionId.Value)
	assert.Equal(fmt.Sprintf("%d", d.IMEI), n.Imei.Value)
	assert.Equal(fmt.Sprintf("%d", d.IMSI), n.Imsi.Value)
	assert.Equal(d.Firmware.CurrentFirmwareID.String(), n.Firmware.CurrentFirmwareId.Value)
	assert.Equal(d.Firmware.TargetFirmwareID.String(), n.Firmware.TargetFirmwareId.Value)
	assert.Equal(d.Firmware.SerialNumber, n.Firmware.SerialNumber.Value)
	assert.Equal(d.Firmware.ModelNumber, n.Firmware.ModelNumber.Value)
	assert.Equal(d.Firmware.Manufacturer, n.Firmware.Manufacturer.Value)
	assert.Equal(d.Firmware.FirmwareVersion, n.Firmware.FirmwareVersion.Value)
	assert.Equal("Downloading", n.Firmware.State.Value)
	assert.Equal(d.Firmware.StateMessage, n.Firmware.StateMessage.Value)
	assert.Equal(d.Network.AllocatedAt.UnixNano()/int64(time.Millisecond), n.Network.AllocatedAt.Value)
	assert.Equal(d.Network.AllocatedIP, n.Network.AllocatedIp.Value)
	assert.Equal(d.Network.CellID, n.Network.CellId.Value)

	// use collection management - should pick up the collection settings here
	c.Firmware.Management = model.CollectionManagement
	c.Firmware.CurrentFirmwareID = model.FirmwareKey(3)
	c.Firmware.TargetFirmwareID = model.FirmwareKey(4)
	n = NewDeviceFromModel(d, c)
	assert.Equal(c.Firmware.CurrentFirmwareID.String(), n.Firmware.CurrentFirmwareId.Value)
	assert.Equal(c.Firmware.TargetFirmwareID.String(), n.Firmware.TargetFirmwareId.Value)

	// Apply field masks to conversion
	c.FieldMask = model.IMEIMask | model.IMSIMask | model.LocationMask

	n = NewDeviceFromModel(d, c)
	assert.Nil(n.Imei)
	assert.Nil(n.Imsi)
	assert.Nil(n.Network.CellId)

	// ...Firmware states
	d.Firmware.State = model.Current
	n = NewDeviceFromModel(d, c)
	assert.Equal("Current", n.Firmware.State.Value)

	d.Firmware.State = model.Unknown
	n = NewDeviceFromModel(d, c)
	assert.Equal("Current", n.Firmware.State.Value)

	d.Firmware.State = model.Initializing
	n = NewDeviceFromModel(d, c)
	assert.Equal("Initializing", n.Firmware.State.Value)

	d.Firmware.State = model.Pending
	n = NewDeviceFromModel(d, c)
	assert.Equal("Pending", n.Firmware.State.Value)

	d.Firmware.State = model.Downloading
	n = NewDeviceFromModel(d, c)
	assert.Equal("Downloading", n.Firmware.State.Value)

	d.Firmware.State = model.Completed
	n = NewDeviceFromModel(d, c)
	assert.Equal("Completed", n.Firmware.State.Value)

	d.Firmware.State = model.UpdateFailed
	n = NewDeviceFromModel(d, c)
	assert.Equal("UpdateFailed", n.Firmware.State.Value)

	d.Firmware.State = model.TimedOut
	n = NewDeviceFromModel(d, c)
	assert.Equal("TimedOut", n.Firmware.State.Value)

	d.Firmware.State = model.Reverted
	n = NewDeviceFromModel(d, c)
	assert.Equal("Reverted", n.Firmware.State.Value)
}

func TestOutputConversion(t *testing.T) {
	assert := require.New(t)

	o := model.NewOutput()
	o.SetTag("name", "Some name")
	o.CollectionID = model.CollectionKey(1)
	o.ID = model.OutputKey(2)
	o.Enabled = true
	o.Type = "udp"
	o.CollectionFieldMask = model.FieldMask(0)
	o.Config = map[string]interface{}{
		outputconfig.UDPHost: "1.2.3.4",
		outputconfig.UDPPort: float64(4711), // JSON structs end up as float64
	}

	n := NewOutputFromModel(o)
	assert.Equal(o.ID.String(), n.OutputId.Value)
	assert.Equal(o.CollectionID.String(), n.CollectionId.Value)
	assert.Equal(apipb.Output_udp, n.Type)
	assert.Equal(o.Config[outputconfig.UDPHost], n.Config.Host.Value)
	assert.Equal(o.Config[outputconfig.UDPPort], float64(n.Config.Port.Value))

	o.Type = "webhook"
	o.Config = map[string]interface{}{
		outputconfig.WebhookURLField:          "http://example.com/webhook",
		outputconfig.WebhookBasicAuthUser:     "johndoe",
		outputconfig.WebhookBasicAuthPass:     "password",
		outputconfig.WebhookCustomHeaderName:  "secrets-header",
		outputconfig.WebhookCustomHeaderValue: "secret-value",
	}

	n = NewOutputFromModel(o)
	assert.Equal(apipb.Output_webhook, n.Type)
	assert.Equal(o.Config[outputconfig.WebhookURLField], n.Config.Url.Value)
	assert.Equal(o.Config[outputconfig.WebhookBasicAuthUser], n.Config.BasicAuthUser.Value)
	assert.Equal(o.Config[outputconfig.WebhookBasicAuthPass], n.Config.BasicAuthPass.Value)
	assert.Equal(o.Config[outputconfig.WebhookCustomHeaderName], n.Config.CustomHeaderName.Value)
	assert.Equal(o.Config[outputconfig.WebhookCustomHeaderValue], n.Config.CustomHeaderValue.Value)

	o.Type = "mqtt"
	o.Config = map[string]interface{}{
		outputconfig.MQTTClientID:         "clientid",
		outputconfig.MQTTDisableCertCheck: true,
		outputconfig.MQTTEndpoint:         "tcp://example.com",
		outputconfig.MQTTPassword:         "secret",
		outputconfig.MQTTTopicName:        "topic",
		outputconfig.MQTTUsername:         "user",
	}
	n = NewOutputFromModel(o)
	assert.Equal(apipb.Output_mqtt, n.Type)
	assert.Equal(o.Config[outputconfig.MQTTClientID], n.Config.ClientId.Value)
	assert.Equal(o.Config[outputconfig.MQTTDisableCertCheck], n.Config.DisableCertCheck.Value)
	assert.Equal(o.Config[outputconfig.MQTTTopicName], n.Config.TopicName.Value)
	assert.Equal(o.Config[outputconfig.MQTTEndpoint], n.Config.Endpoint.Value)
	assert.Equal(o.Config[outputconfig.MQTTUsername], n.Config.Username.Value)
	assert.Equal(o.Config[outputconfig.MQTTPassword], n.Config.Password.Value)

	o.Type = "ifttt"
	o.Config = map[string]interface{}{
		outputconfig.IFTTTKey:        "key",
		outputconfig.IFTTTEvent:      "event",
		outputconfig.FTTTAsIsPayload: true,
	}
	n = NewOutputFromModel(o)
	assert.Equal(apipb.Output_ifttt, n.Type)
	assert.Equal(o.Config[outputconfig.IFTTTKey], n.Config.Key.Value)
	assert.Equal(o.Config[outputconfig.IFTTTEvent], n.Config.EventName.Value)
	assert.Equal(o.Config[outputconfig.FTTTAsIsPayload], n.Config.AsIsPayload.Value)
}

func TestTeamConversion(t *testing.T) {
	assert := require.New(t)

	team := model.NewTeam()
	team.ID = model.TeamKey(100)
	team.SetTag("foo", "bar")
	team.SetTag("bar", "baz")
	team.SetTag("baz", "foo")

	user1 := model.NewUser(1, "1", model.AuthConnectID, 1)
	user2 := model.NewUser(2, "2", model.AuthGitHub, 2)
	team.AddMember(model.NewMember(user1, model.AdminRole))
	team.AddMember(model.NewMember(user2, model.MemberRole))

	apiTeam := NewTeamFromModel(team, true)
	assert.Equal(team.ID.String(), apiTeam.TeamId.Value)

	assert.Len(apiTeam.Members, 2)

	// Ensure the map is copied properly
	team.SetTag("foo", "boo")
	team.SetTag("bar", "boo")
	team.SetTag("baz", "boo")
	assert.Equal("bar", apiTeam.Tags["foo"])
	assert.Equal("baz", apiTeam.Tags["bar"])
	assert.Equal("foo", apiTeam.Tags["baz"])
}

func TestTokenConversion(t *testing.T) {
	assert := require.New(t)

	token := model.NewToken()
	token.GenerateToken()
	token.Write = true
	token.Resource = "something"
	token.SetTag("name", "value")

	apiToken := NewTokenFromModel(token)
	assert.True(apiToken.Write.Value)
	assert.Equal("something", apiToken.Resource.Value)
	assert.Contains(apiToken.Tags, "name")
	assert.Equal("value", apiToken.Tags["name"])
}
func TestInviteConversion(t *testing.T) {
	assert := require.New(t)

	invite, err := model.NewInvite(1, 1)
	assert.NoError(err)
	i := NewInviteFromModel(invite)
	assert.Equal(invite.Code, i.Code.Value)
	assert.Equal(invite.Created.UnixNano()/int64(time.Millisecond), i.CreatedAt.Value)

}
func TestUserConversion(t *testing.T) {
	assert := require.New(t)

	user := model.NewUser(1, "external", model.AuthConnectID, 1)
	user.Name = "name"
	user.Email = "email"
	user.Phone = "phone"
	user.VerifiedEmail = true
	user.VerifiedPhone = false
	user.AvatarURL = "avatar"

	p := NewUserProfileFromUser(user)
	assert.Equal("name", p.Name.Value)
	assert.Equal("email", p.Email.Value)
	assert.Equal("phone", p.Phone.Value)
	assert.Equal("avatar", p.AvatarUrl.Value)
	assert.Equal("external", p.ConnectId.Value)
	assert.Equal("connect", p.Provider.Value)

	user.AuthType = model.AuthGitHub
	p = NewUserProfileFromUser(user)
	assert.Equal("external", p.GithubLogin.Value)
	assert.Equal("github", p.Provider.Value)

	user.AuthType = model.AuthInternal
	p = NewUserProfileFromUser(user)
	assert.Equal("internal", p.Provider.Value)

	user.AuthType = model.AuthMethod(999)
	p = NewUserProfileFromUser(user)
	assert.Nil(p.Provider)
}

func TestDownstreamMessageConversion(t *testing.T) {
	assert := require.New(t)

	// Nil request => error
	_, err := NewDownstreamMessage(nil)
	assert.Error(err)

	// No fields at all => error
	r := &apipb.SendMessageRequest{}
	_, err = NewDownstreamMessage(r)
	assert.Error(err)

	// Missing payload
	r.Port = &wrappers.Int32Value{Value: 4711}
	_, err = NewDownstreamMessage(r)
	assert.Error(err)

	// Just payload and port should get us a valid UDP message
	r.Payload = []byte("hello")
	res, err := NewDownstreamMessage(r)
	assert.NoError(err)
	assert.Equal(model.UDPTransport, res.Transport)
	assert.Equal(4711, res.Port)
	assert.Equal([]byte("hello"), res.Payload)

	// Unknown transport
	r.Transport = &wrappers.StringValue{Value: "carrier-pigeon"}
	_, err = NewDownstreamMessage(r)
	assert.Error(err)

	// CoAP messages need path but not port
	r.Port = nil
	r.CoapPath = &wrappers.StringValue{Value: "/something"}
	r.Transport = &wrappers.StringValue{Value: "coap"}
	res, err = NewDownstreamMessage(r)
	assert.NoError(err)
	assert.Equal(model.CoAPTransport, res.Transport)
	assert.Equal(5683, res.Port)
	assert.Equal([]byte("hello"), res.Payload)

	// Omit path => error
	r.CoapPath = nil
	_, err = NewDownstreamMessage(r)
	assert.Error(err)
}

func TestApplyDataFilter(t *testing.T) {
	assert := require.New(t)

	r := &apipb.ListMessagesRequest{}
	df := &datastore.DataFilter{}

	ApplyDataFilter(r, df)
	assert.Equal(df.Limit, int32(collectionDataItemLimit))

	start := time.Now()
	until := start.Add(10 * time.Minute)

	r.Limit = &wrappers.Int32Value{Value: 1000}
	r.Since = &wrappers.Int64Value{Value: start.UnixNano() / int64(time.Millisecond)}
	r.Until = &wrappers.Int64Value{Value: until.UnixNano() / int64(time.Millisecond)}

	ApplyDataFilter(r, df)

	assert.Equal(df.Limit, int32(1000))
	assert.Equal(df.From, start.UnixNano()/int64(time.Millisecond)*int64(time.Millisecond))
	assert.Equal(df.To, until.UnixNano()/int64(time.Millisecond)*int64(time.Millisecond))
}

func TestLogConversion(t *testing.T) {
	assert := require.New(t)

	logs := []model.OutputLogEntry{
		model.OutputLogEntry{Message: "M1", Time: time.Now(), Repeated: 1},
		model.OutputLogEntry{Message: "M2", Time: time.Now(), Repeated: 2},
		model.OutputLogEntry{Message: "M3", Time: time.Now(), Repeated: 3},
		model.OutputLogEntry{Message: "M4", Time: time.Now(), Repeated: 4},
	}
	assert.Nil(NewOutputLogsFromModel(nil))

	ret := NewOutputLogsFromModel(logs)
	assert.NotNil(ret)
	assert.Len(ret, len(logs))
}

func TestStatusConversion(t *testing.T) {
	assert := require.New(t)
	ret := NewOutputStatusFromModel("1", "2", true, model.OutputStatus{Forwarded: 1, Received: 2, ErrorCount: 3, Retransmits: 4})
	assert.Equal(ret.CollectionId.Value, "1")
	assert.Equal(ret.OutputId.Value, "2")
	assert.Equal(ret.Enabled.Value, true)
	assert.Equal(ret.Forwarded.Value, int32(1))
	assert.Equal(ret.Received.Value, int32(2))
	assert.Equal(ret.ErrorCount.Value, int32(3))
	assert.Equal(ret.Retransmits.Value, int32(4))
}

func TestConfigFromAPIConversion(t *testing.T) {
	assert := require.New(t)
	cfg := NewOutputConfigFromAPI(&apipb.Output{})
	assert.Empty(cfg)

	cfg = NewOutputConfigFromAPI(&apipb.Output{
		Type: apipb.Output_udp,
		Config: &apipb.OutputConfig{
			AsIsPayload: &wrappers.BoolValue{Value: true},
			EventName:   &wrappers.StringValue{Value: "name"},
			Key:         &wrappers.StringValue{Value: "key"},
		},
	})
	assert.True(cfg[outputconfig.FTTTAsIsPayload].(bool))
	assert.Equal(cfg[outputconfig.IFTTTEvent].(string), "name")
	assert.Equal(cfg[outputconfig.IFTTTKey].(string), "key")

	cfg = NewOutputConfigFromAPI(&apipb.Output{
		Type: apipb.Output_webhook,
		Config: &apipb.OutputConfig{
			Url:               &wrappers.StringValue{Value: "url"},
			BasicAuthPass:     &wrappers.StringValue{Value: "bap"},
			BasicAuthUser:     &wrappers.StringValue{Value: "bau"},
			CustomHeaderName:  &wrappers.StringValue{Value: "chn"},
			CustomHeaderValue: &wrappers.StringValue{Value: "chv"},
		},
	})
	assert.Equal(cfg[outputconfig.WebhookBasicAuthPass].(string), "bap")
	assert.Equal(cfg[outputconfig.WebhookBasicAuthUser].(string), "bau")
	assert.Equal(cfg[outputconfig.WebhookURLField].(string), "url")
	assert.Equal(cfg[outputconfig.WebhookCustomHeaderName].(string), "chn")
	assert.Equal(cfg[outputconfig.WebhookCustomHeaderValue].(string), "chv")

	cfg = NewOutputConfigFromAPI(&apipb.Output{
		Type: apipb.Output_udp,
		Config: &apipb.OutputConfig{
			Host: &wrappers.StringValue{Value: "host"},
			Port: &wrappers.Int32Value{Value: 8080},
		},
	})
	assert.Equal(cfg[outputconfig.UDPHost].(string), "host")
	assert.Equal(cfg[outputconfig.UDPPort].(float64), float64(8080.0))

	cfg = NewOutputConfigFromAPI(&apipb.Output{
		Type: apipb.Output_mqtt,
		Config: &apipb.OutputConfig{
			ClientId:         &wrappers.StringValue{Value: "clientid"},
			DisableCertCheck: &wrappers.BoolValue{Value: true},
			Endpoint:         &wrappers.StringValue{Value: "ep"},
			Password:         &wrappers.StringValue{Value: "pass"},
			TopicName:        &wrappers.StringValue{Value: "topic"},
			Username:         &wrappers.StringValue{Value: "user"},
		},
	})
	assert.Equal(cfg[outputconfig.MQTTClientID].(string), "clientid")
	assert.Equal(cfg[outputconfig.MQTTDisableCertCheck].(bool), true)
	assert.Equal(cfg[outputconfig.MQTTEndpoint].(string), "ep")
	assert.Equal(cfg[outputconfig.MQTTPassword].(string), "pass")
	assert.Equal(cfg[outputconfig.MQTTTopicName].(string), "topic")
	assert.Equal(cfg[outputconfig.MQTTUsername].(string), "user")
}
