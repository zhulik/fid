package elect

type ElectionStatus int

const (
	Won ElectionStatus = iota
	Lost
	Error
	Cancelled
)

type Outcome struct {
	Status ElectionStatus
	Error  error // only set if Status is Error or Cancelled
}
