package elect

type ElectionStatus int32

const (
	Unknown ElectionStatus = iota
	Won
	Lost
	Error
	Stopped
)

type Outcome struct {
	Status ElectionStatus
	Error  error // only set if Status is Error or Stopped
}
