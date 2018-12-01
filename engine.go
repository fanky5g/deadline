package deadline

import (
	"context"
	"flag"
	"fmt"
)

// Engine is the main construct that runs in background to check which items
// have timedout and executes corresponding instruction
type Engine struct {
	running       bool
	ticked        float64
	quit          chan int
	pool          chan Contract
	log           map[string]map[context.Context]context.CancelFunc //shallow log of entries
	memoryStorage map[string]Contract
	store         Store

	workerQueue chan Worker
	done        chan string
	delete      map[string]bool
}

// Serializer takes a byte array and returns a Contract map
type Serializer func([]byte) (map[string]Contract, error)

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
func New(ctx context.Context, numWorkers int, store Store) (*Engine, error) {
	// make a buffered channel of contracts
	engine := new(Engine)
	engine.quit = make(chan int)
	engine.pool = make(chan Contract, poolBuffer)
	engine.log = make(map[string]map[context.Context]context.CancelFunc)
	engine.memoryStorage = make(map[string]Contract)
	engine.workerQueue = make(chan Worker, numWorkers)
	engine.done = make(chan string)
	engine.delete = make(map[string]bool)
	engine.ticked = 0
	engine.store = store

	go func() {
		<-ctx.Done()
		engine.quit <- 1
	}()

	for i := 0; i < numWorkers; i++ {
		go newWorker(i+1, engine.workerQueue)
	}

	if store != nil {
		hist, err := store.Load()
		if err != nil {
			return nil, err
		}

		if len(hist) > 0 {
			engine.memoryStorage = hist
			for k, v := range hist {
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
				if _, ok := engine.delete[cont.ID]; !ok {
					// get next free worker
					worker := <-engine.workerQueue

					if contextMap, ok := engine.log[cont.ID]; ok {
						for hotExit := range contextMap {
							worker.Start(c, hotExit, _heartBeat, cont, engine.done)
						}
					} else {
						worker.Start(c, nil, _heartBeat, cont, engine.done)
					}
				} else {
					delete(engine.delete, cont.ID)
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
	if engine.store != nil {
		if err := engine.store.Clear(); err != nil {
			return err
		}

		if err := engine.store.Save(engine.memoryStorage); err != nil {
			return err
		}
	}

	return nil
}

func cleanup(engine *Engine) {
	engine.log = nil
	engine.memoryStorage = nil
	engine.pool = nil
	engine.ticked = 0
	engine.quit = nil
	engine.running = false
	engine.workerQueue = nil
	engine.done = nil
	engine = nil
}

// ClearStorage empties all contents of the storage file
func (engine *Engine) clearStorage() error {
	if engine.store != nil {
		return engine.store.Clear()
	}
	return nil
}

// Enqueue adds a new entity contract pool to engine
func (engine *Engine) Enqueue(contract Contract) error {
	// check if item already exists in pool
	id := contract.ID
	if _, ok := engine.log[id]; ok {
		return fmt.Errorf("cannot enqueue item. entry with id %v already exists", id)
	}

	engine.LogContext(contract.ID)
	engine.memoryStorage[id] = contract
	engine.pool <- contract

	if engine.store != nil {
		return engine.store.Enqueue(contract)
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
