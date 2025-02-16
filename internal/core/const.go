package core

import (
	"time"
)

const (
	InvocationStreamName = "INVOCATION" // used as INVOCATION:<function_name>

	RequestIDHeaderName       = "Lambda-Runtime-Aws-Request-Id"
	RequestDeadlineHeaderName = "Lambda-Runtime-Deadline-Ms"

	LabelNameComponent    = "wtf.zhulik.fid.component"
	LabelNameMaxScale     = "wtf.zhulik.fid.scale.max"
	LabelNameMinScale     = "wtf.zhulik.fid.scale.min"
	LabelNameTimeout      = "wtf.zhulik.fid.timeout"
	LabelNameFunctionName = "wtf.zhulik.fid.name"

	FunctionComponentLabelValue         = "function"
	FunctionTemplateComponentLabelValue = "function-template"

	ContentTypeJSON = "application/json; charset=utf-8"

	MaxTimeout      = 15 * time.Minute
	DefaultTimeout  = 10 * time.Second
	DefaultMinScale = 1
	DefaultMaxScale = 10

	ForwarderImageName = "ghcr.io/zhulik/fid-forwarder"
)
