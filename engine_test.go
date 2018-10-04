package deadline

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var (
	_engine   *Engine
	_quitChan = make(chan int, 1)
)

func TestMain(m *testing.M) {
	filePath, err := filepath.Abs("./store")
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}

	deadline, err := New(10, filePath, _quitChan, func(data json.RawMessage) (map[string]Contract, error) {
		jsonString, err := data.MarshalJSON()
		if err != nil {
			return nil, err
		}

		decoded := make(map[string]c)
		if err := json.Unmarshal(jsonString, &decoded); err != nil {
			return nil, err
		}

		out := make(map[string]Contract)
		for k, v := range decoded {
			out[k] = &v
		}

		return out, nil
	})

	if err != nil {
		log.Fatalf("Expected error to be nil but got: %v", err)
	}

	_engine = deadline
	_engine.Start()

	code := m.Run()
	os.Exit(code)
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

func TestTimeIsPast(t *testing.T) {
	refTime := time.Now().Unix()
	<-time.After(time.Microsecond * 2)
	isPast := IsUnixTimePast(refTime)
	if isPast != true {
		t.Log("Expected time past to be false")
		t.Fail()
	}
}

func TestEnqueue(t *testing.T) {
	if len(_engine.log) < 5 {
		rand.Seed(time.Now().UTC().UnixNano())
		contract := &c{
			ID:      fmt.Sprintf("%v", randInt(1, 10000000)),
			Timeout: time.Now().Add(time.Second * 10).Unix(),
		}

		err := _engine.Enqueue(contract)
		if err != nil {
			t.Errorf("Failed to enqueue new contract: %v", err)
		}

		if _, ok := _engine.log[contract.GetIdentifier()]; !ok {
			t.Errorf("Failed to save entry log")
		}

		if _, ok := _engine.memoryStorage[contract.GetIdentifier()]; !ok {
			t.Errorf("Failed to save entry in memory")
		}
	}

	// wait around for os.Signal
	// var stopChan = make(chan os.Signal, 2)
	// signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// <-stopChan
	// _quitChan <- 1
	// <-time.After(time.Second * 3)
}
