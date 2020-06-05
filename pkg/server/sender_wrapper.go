package server

//
//Copyright 2019 Telenor Digital AS
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/apn"
	"github.com/eesrc/horde/pkg/deviceio/rxtx"
	"github.com/eesrc/horde/pkg/metrics"
	"github.com/eesrc/horde/pkg/model"
	"github.com/go-ocf/go-coap"
	"github.com/go-ocf/go-coap/codes"
)

const sendTimeoutSeconds = 10

// messageSender is a wrapper for the apnServers Send method
// that increments the performance counters when a message is
// sent
type messageSender struct {
	rxtxReceiver *apn.RxTxReceiver
}

func (m *messageSender) Send(device model.Device, msg model.DownstreamMessage) error {
	txmsg := &rxtx.Message{
		Payload:       msg.Payload,
		RemoteAddress: net.ParseIP(device.Network.AllocatedIP),
		RemotePort:    int32(msg.Port),
	}
	switch msg.Transport {
	case model.CoAPPullTransport:
		txmsg.Type = rxtx.MessageType_CoAPPull
		txmsg.Coap = &rxtx.CoAPOptions{
			Path:           msg.Path,
			Code:           int32(codes.POST),
			ContentFormat:  int32(coap.AppOctets),
			TimeoutSeconds: int32(sendTimeoutSeconds),
		}
	case model.CoAPTransport:

		txmsg.Type = rxtx.MessageType_CoAPPush
		txmsg.Coap = &rxtx.CoAPOptions{
			Path:           msg.Path,
			Code:           int32(codes.POST),
			ContentFormat:  int32(coap.AppOctets),
			TimeoutSeconds: int32(sendTimeoutSeconds),
		}
	case model.UDPTransport:
		txmsg.Type = rxtx.MessageType_UDP
		txmsg.Udp = &rxtx.UDPOptions{}
	default:
		logging.Error("Don't know how to process %v transport", msg.Transport)
		return errors.New("unknown transport")
	}

	ctx, done := context.WithTimeout(context.Background(), time.Duration(sendTimeoutSeconds)*time.Second)
	defer done()

	code, err := m.rxtxReceiver.Send(ctx, device, txmsg, true)
	if code == rxtx.ErrorCode_SUCCESS || code == rxtx.ErrorCode_PENDING {
		metrics.DefaultCoreCounters.MessagesOutCount.Add(1)
		return nil
	}
	logging.Info("Got error sending message: %v (%s) (dest address: %s, port: %d, IMSI: %d)",
		err, code.String(), device.Network.AllocatedIP, msg.Port, device.IMSI)
	return errors.New(code.String())
}
