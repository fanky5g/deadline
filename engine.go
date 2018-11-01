package deadline

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

// Engine is the main construct that runs in background to check which items
// have timedout and executes corresponding instruction
type Engine struct {
	running       bool
	ticked        float64
	quit          chan int
	pool          chan Contract
	log           map[string]map[context.Context]context.CancelFunc //shallow log of entries
	fileStorage   *os.File
	memoryStorage map[string]Contract

	workerQueue chan Worker
	done        chan string
	delete      map[string]bool
}

type decodeFunc func(json.RawMessage) (map[string]Contract, error)

var (
	poolBuffer int
	_heartBeat float64
)

func init() {
	// create new engine object
	flag.Float64Var(&_heartBeat, "heartbeat", 1000, "interval in ms for engine execution")
	flag.IntVar(&poolBuffer, "pool", 100, "pool buffer")
}

// New creates a new engine object
// @param numWorkers: number of workers to create
// @return *Engine
func New(numWorkers int, filepath string, quit chan int, decode decodeFunc) (*Engine, error) {
	// make a buffered channel of contracts
	engine := new(Engine)
	engine.quit = quit
	engine.pool = make(chan Contract, poolBuffer)
	engine.log = make(map[string]map[context.Context]context.CancelFunc)
	engine.memoryStorage = make(map[string]Contract)
	engine.workerQueue = make(chan Worker, numWorkers)
	engine.done = make(chan string)
	engine.delete = make(map[string]bool)
	engine.ticked = 0

	for i := 0; i < numWorkers; i++ {
		go newWorker(i+1, engine.workerQueue)
	}

	// open file for reading and return error if fails
	if filepath != "" {
		f, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}

		// check if f has contents and load them into engine
		engine.fileStorage = f
		data, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, err
		}

		if len(data) > 0 {
			decoded, err := decode(json.RawMessage(data))
			if err != nil {
				return nil, err
			}

			// we need to start processing engine right away so we can make sure buffer doesn't get blocked
			engine.Start()
			engine.memoryStorage = decoded
			for k, v := range decoded {
				engine.pool <- v
				engine.LogContext(k)
			}
		}
	}

	return engine, nil
}

// Start starts a non-blocking execution of engine
func (engine *Engine) Start() {
	if engine.running {
		return
	}

	engine.running = true
	c, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			select {
			// receive new contract
			case cont := <-engine.pool:
				if _, ok := engine.delete[cont.GetIdentifier()]; !ok {
					// get next free worker
					worker := <-engine.workerQueue

					if contextMap, ok := engine.log[cont.GetIdentifier()]; ok {
						for hotExit := range contextMap {
							worker.Start(c, hotExit, _heartBeat, cont, engine.done)
						}
					} else {
						worker.Start(c, nil, _heartBeat, cont, engine.done)
					}
				} else {
					delete(engine.delete, cont.GetIdentifier())
				}
			case id := <-engine.done:
				delete(engine.log, id)
				delete(engine.memoryStorage, id)
				engine.saveSnapshot()
			}
		}
	}()

	go func() {
		<-engine.quit
		cancel()
		cleanup(engine)
	}()
}

func (engine *Engine) saveSnapshot() error {
	data := &bytes.Buffer{}

	if err := json.NewEncoder(data).Encode(engine.memoryStorage); err != nil {
		return err
	}

	if err := engine.ClearStorage(); err != nil {
		return err
	}

	_, err := engine.fileStorage.WriteString(data.String())
	if err != nil {
		return err
	}

	if err := engine.fileStorage.Close(); err != nil {
		return err
	}
	return nil
}

func cleanup(engine *Engine) {
	engine.log = nil
	engine.fileStorage = nil
	engine.memoryStorage = nil
	engine.pool = nil
	engine.ticked = 0
	engine.fileStorage = nil
	engine.quit = nil
	engine.running = false
	engine.workerQueue = nil
	engine.done = nil
	engine = nil
}

// ClearStorage empties all contents of the storage file
func (engine *Engine) ClearStorage() error {
	engine.fileStorage.Truncate(0)
	return nil
}

// Enqueue adds a new entity contract pool to engine
func (engine *Engine) Enqueue(contract Contract) error {
	// check if item already exists in pool
	id := contract.GetIdentifier()
	if _, ok := engine.log[id]; ok {
		return fmt.Errorf("cannot enqueue item. entry with id %v already exists", id)
	}

	engine.LogContext(contract.GetIdentifier())
	engine.memoryStorage[id] = contract
	engine.pool <- contract

	if engine.fileStorage != nil {
		return engine.saveSnapshot()
	}

	return nil
}

// LogContext adds cancellable context to log
func (engine *Engine) LogContext(contractID string) {
	c, cancel := context.WithCancel(context.Background())

	engine.log[contractID] = map[context.Context]context.CancelFunc{
		c: cancel,
	}
}

// Prune remove contract from engine pool, if worker already spawned for contract, close worker
func (engine *Engine) Prune(contractID string) {
	if contextMap, ok := engine.log[contractID]; ok {
		for _, v := range contextMap {
			v()
		}
	}

	// mark for deletion
	engine.delete[contractID] = true
}
