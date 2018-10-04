package deadline

// Contract represents a unit entity to keep in check
// It defines methods GetIdentifier to get identifier of type
// CheckTimeout checks if the entity's timeout has been reached
// ExecOnTimeout executes specific instructions after timeout has been reached
type Contract interface {
	GetIdentifier() string
	CheckTimeout() bool
	ExecOnTimeout() error
	LogError(...interface{})
}
