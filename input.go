package deadline

import (
	"fmt"
	"log"
)

// example input type...
type c struct {
	ID      string
	Timeout int64
}

func (item *c) GetIdentifier() string {
	return item.ID
}

func (item *c) CheckTimeout() bool {
	return false
}

func (item *c) ExecOnTimeout() error {
	fmt.Printf("Item %v timed out.\n", item.ID)
	return nil
}

func (item *c) LogError(v ...interface{}) {
	log.Println(v)
}
