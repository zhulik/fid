package core

type ScalingRequestType int

const (
	ScalingRequestTypeScaleUp   ScalingRequestType = iota
	ScalingRequestTypeScaleDown ScalingRequestType = iota
)

type ScalingRequest struct {
	Type ScalingRequestType

	InstanceIDs []string // Should be specified for ScaleDown requests
	Count       int      // Should be specified for ScaleUp requests
}
