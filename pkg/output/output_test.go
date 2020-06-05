package output

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
	"math/rand"
	"testing"
	"time"

	"github.com/eesrc/horde/pkg/model"
)

func outputBenchmark(output Output, config model.OutputConfig, b *testing.B) {
	if _, err := output.Validate(config); err != nil {
		b.Fatal("Invalid configuration: ", err)
	}

	dataChan := make(chan interface{})

	output.Start(config, 0, 0, dataChan)

	for i := 0; i < b.N; i++ {
		data := make([]byte, rand.Int31n(45))
		device := model.NewDevice()
		device.IMEI = rand.Int63()
		device.IMSI = rand.Int63()
		msg := model.DataMessage{Payload: data, Device: device, Received: time.Now()}
		select {
		case dataChan <- msg:
			// ok - continue
		case <-time.After(15 * time.Millisecond):
			b.Fatalf("Output is lagging behind (%d elements sent so far)", i)
		}
	}

	time.Sleep(10 * time.Millisecond)
	status := output.Status()

	if status.Received != b.N {
		b.Fatalf("Output haven't received all messages. Expected %d but got %d", b.N, status.Received)
	}

}

func outputTests(output Output, config model.OutputConfig, t *testing.T) {
	output.Logs()

	if errs, err := output.Validate(config); err != nil {
		t.Fatalf("Invalid configuration: %+v: %v", errs, err)
	}

	const msgCount = 10

	dataChan := make(chan interface{})

	output.Logs()

	output.Start(config, 0, 0, dataChan)

	for i := 0; i < msgCount; i++ {
		data := make([]byte, rand.Int31n(45))
		device := model.NewDevice()
		device.IMEI = rand.Int63()
		device.IMSI = rand.Int63()
		msg := model.DataMessage{Payload: data, Device: device, Received: time.Now()}
		select {
		case dataChan <- msg:
			// ok - continue
		case <-time.After(15 * time.Millisecond):
			t.Fatalf("Output is lagging behind (%d elements sent so far)", i)
		}
	}

	time.Sleep(10 * time.Millisecond)
	status := output.Status()

	if status.Received != msgCount {
		t.Fatalf("Output haven't received all messages. Expected %d but got %d", msgCount, status.Received)
	}

	if status.ErrorCount > msgCount/100 {
		t.Fatalf("Error ratio is > 1%%. ErrorsCount = %d", status.ErrorCount)
	}

	// Stop then start output
	output.Stop(100 * time.Millisecond)

	output.Start(config, 0, 0, dataChan)

	for i := 0; i < msgCount*2; i++ {
		data := make([]byte, rand.Int31n(45))
		device := model.NewDevice()
		device.IMEI = rand.Int63()
		device.IMSI = rand.Int63()
		msg := model.DataMessage{Payload: data, Device: device, Received: time.Now()}
		select {
		case dataChan <- msg:
			// ok - continue
		case <-time.After(15 * time.Millisecond):
			t.Fatalf("Output is lagging behind. %d messages sent so far", i)
		}
	}
	output.Logs()

	close(dataChan)
	output.Stop(100 * time.Millisecond)
}
