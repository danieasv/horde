package deviceio

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
	"context"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/deviceio/rxtx"
)

// Common functions and constants for the listeners

// The default grpcTimeout is quite short. If it fails it should fail reasonably
// quickly.
const grpcTimeout = time.Millisecond * 500

// Retry is a bit longer since there might be a cluster resharding in the other
// end.
const grpcRetryTimeout = time.Second * 5

// Time to sleep when polling the upstream service yields an error
const sleepOnError = 1 * time.Second

// Time to sleep when the upstream service returns an empty response.
const sleepOnEmpty = 100 * time.Millisecond

// Send ack on message to the upstream service. There's a retry if the initial call.
func sendAckWithRetry(client rxtx.RxtxClient, messageID int64, errorCode rxtx.ErrorCode) {
	ctx, done := context.WithTimeout(context.Background(), grpcTimeout)
	defer done()

	req := &rxtx.AckRequest{
		MessageId: messageID,
		Result:    errorCode,
	}
	_, err := client.Ack(ctx, req)
	if err != nil {
		ctx2, done2 := context.WithTimeout(context.Background(), grpcRetryTimeout)
		defer done2()
		_, err = client.Ack(ctx2, req)
	}
	if err != nil {
		logging.Error("Could not ack response (message id=%d, code=%s) to upstream server: %v", messageID, errorCode.String(), err)
	}
}
