package internal

type ChangeEvent[T any] struct {
	Operation Operation
	// ID of the document.
	ID string
	// Data is nil if operation is delete
	Data *T
}

type Operation string

const (
	Insert Operation = "insert"
	Update Operation = "update"
	Delete Operation = "delete"
)
