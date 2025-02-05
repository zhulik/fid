package elect

type ElectionStatus int

const (
	Unknown ElectionStatus = iota
	Won
	Lost
	Error
	Cancelled
)

type Outcome struct {
	Status ElectionStatus
	Error  error // only set if Status is Error or Cancelled
}
