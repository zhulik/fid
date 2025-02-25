package core

import (
	"time"
)

const (
	InvocationStreamName = "INVOCATION" // used as INVOCATION:<function_name>

	RequestIDHeaderName       = "Lambda-Runtime-Aws-Request-Id"
	RequestDeadlineHeaderName = "Lambda-Runtime-Deadline-Ms"

	LabelNameComponent = "wtf.zhulik.fid.component"

	// TODO: unify naming with LabelName*.
	RuntimeAPIComponentLabelValue = "runtimeapi"
	FunctionComponentLabelValue   = "function"
	ScalerComponentLabelValue     = "scaler"
	InfoServerComponentLabelValue = "info-server"
	GatewayComponentLabelValue    = "gateway"

	ContentTypeJSON = "application/json; charset=utf-8"

	MaxTimeout      = 15 * time.Minute
	DefaultTimeout  = 10 * time.Second
	DefaultMinScale = 1
	DefaultMaxScale = 10

	ImageNameRuntimeAPI = "ghcr.io/zhulik/fid-runtimeapi"
	ImageNameScaler     = "ghcr.io/zhulik/fid-scaler"
	ImageNameInfoServer = "ghcr.io/zhulik/fid-infoserver"
	ImageNameGateway    = "ghcr.io/zhulik/fid-gateway"

	EnvNameAWSLambdaRuntimeAPI   = "AWS_LAMBDA_RUNTIME_API"
	EnvNameFunctionName          = "FUNCTION_NAME"
	EnvNameFunctionContainerName = "FUNCTION_CONTAINER_NAME"
	EnvNameInstanceID            = "FUNCTION_INSTANCE_ID"
	EnvNameNatsURL               = "NATS_URL"

	ContainerNameInfoServer = "info-server"
	ContainerNameGateway    = "gateway"

	BucketNameFunctions = "functions"
	BucketNameElections = "elections"
)
