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

	// TODO: unify naming with LabelName*.
	RuntimeAPIComponentLabelValue       = "runtimeapi"
	FunctionComponentLabelValue         = "function"
	FunctionTemplateComponentLabelValue = "function-template"
	ScalerComponentLabelValue           = "scaler"

	ContentTypeJSON = "application/json; charset=utf-8"

	MaxTimeout      = 15 * time.Minute
	DefaultTimeout  = 10 * time.Second
	DefaultMinScale = 1
	DefaultMaxScale = 10

	ImageNameRuntimeAPI = "ghcr.io/zhulik/fid-runtimeapi"
	ImageNameScaler     = "ghcr.io/zhulik/fid-scaler"

	EnvNameAWSLambdaRuntimeAPI = "AWS_LAMBDA_RUNTIME_API"
	EnvNameFunctionName        = "FUNCTION_NAME"
	EnvNameInstanceID          = "FUNCTION_INSTANCE_ID"
	EnvNameNatsURL             = "NATS_URL"
)
