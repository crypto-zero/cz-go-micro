package registry

import "time"

// Watcher is an interface that returns updates
// about services within the registry.
type Watcher interface {
	// Next is a blocking call
	Next() (*Result, error)
	Stop()
}

const (
	ResultActionCreate = "create"
	ResultActionUpdate = "update"
	ResultActionDelete = "delete"
)

// Result is returned by a call to Next on
// the watcher. Actions can be created, update, delete
type Result struct {
	Action  string
	Service *Service
}

func (r *Result) IsCreate() bool {
	return r.Action == ResultActionCreate
}

func (r *Result) IsUpdate() bool {
	return r.Action == ResultActionUpdate
}

func (r *Result) IsDelete() bool {
	return r.Action == ResultActionDelete
}

// EventType defines registry event type
type EventType int

const (
	// Create is emitted when a new service is registered
	Create EventType = iota
	// Delete is emitted when an existing service is deregsitered
	Delete
	// Update is emitted when an existing servicec is updated
	Update
)

// String returns human readable event type
func (t EventType) String() string {
	switch t {
	case Create:
		return "create"
	case Delete:
		return "delete"
	case Update:
		return "update"
	default:
		return "unknown"
	}
}

// Event is registry event
type Event struct {
	// Id is registry id
	Id string
	// Type defines type of event
	Type EventType
	// Timestamp is event timestamp
	Timestamp time.Time
	// Service is registry service
	Service *Service
}
