package core

import (
	"time"
)

const (
	StreamNameInvocation = "INVOCATION" // used as INVOCATION:<function_name>

	HeaderNameRequestID       = "Lambda-Runtime-Aws-Request-Id"
	HeaderNameRequestDeadline = "Lambda-Runtime-Deadline-Ms"

	LabelNameComponent = "wtf.zhulik.fid.component"

	ComponentLabelValueRuntimeAPI = "runtimeapi"
	ComponentLabelValueFunction   = "function"
	ComponentLabelValueScaler     = "scaler"
	ComponentLabelValueInfoServer = "info-server"
	ComponentLabelValueGateway    = "gateway"

	ContentTypeJSON = "application/json; charset=utf-8"

	MaxTimeout = 15 * time.Minute

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
