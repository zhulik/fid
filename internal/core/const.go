package core

import (
	"time"
)

const (
	InvocationStreamName = "invocation"

	LabelNameComponent          = "wtf.zhulik.fid.component"
	LabelNameMaxScale           = "wtf.zhulik.fid.scale.max"
	LabelNameMinScale           = "wtf.zhulik.fid.scale.min"
	LabelNameTimeout            = "wtf.zhulik.fid.timeout"
	FunctionComponentLabelValue = "function"

	DefaultTimeout  = 10 * time.Second
	DefaultMinScale = 1
	DefaultMaxScale = 10
)
