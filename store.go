package deadline

// Store is an opt-in backing storage driver implementation
type Store interface {
	// Enqueue adds a contract to the store
	Enqueue(Contract) error
	// Dequeue removes a contract from the store by id
	Dequeue(string) error
	// Load initializes deadline with a history of contracts
	Load() (map[string]Contract, error)
	// Save saves a history of all contracts
	Save(map[string]Contract) error
	// Clear clears the store
	Clear() error
}
