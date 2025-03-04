package core

type ScalingRequestType int

const (
	ScalingRequestTypeScaleUp   ScalingRequestType = iota
	ScalingRequestTypeScaleDown ScalingRequestType = iota
)

type ScalingRequest struct {
	Type ScalingRequestType

	InstanceID string // Should be specified for ScaleDown requests
}
