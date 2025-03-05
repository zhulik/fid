package core

import (
	"time"
)

const (
	StreamNameInvocation = "INVOCATION" // used as INVOCATION:<function_name>

	HeaderNameRequestID       = "Lambda-Runtime-Aws-Request-Id"
	HeaderNameRequestDeadline = "Lambda-Runtime-Deadline-Ms"

	LabelNameComponent = "wtf.zhulik.fid.component"

	ComponentNameRuntimeAPI = "runtimeapi"
	ComponentNameFunction   = "function"
	ComponentNameScaler     = "scaler"
	ComponentNameInfoServer = "info-server"
	ComponentNameGateway    = "gateway"

	ContentTypeJSON = "application/json; charset=utf-8"

	MaxTimeout = 15 * time.Minute

	ImageNameFID = "ghcr.io/zhulik/fid"

	EnvNameAWSLambdaRuntimeAPI   = "AWS_LAMBDA_RUNTIME_API"
	EnvNameFunctionName          = "FUNCTION_NAME"
	EnvNameFunctionContainerName = "FUNCTION_CONTAINER_NAME"
	EnvNameInstanceID            = "FUNCTION_INSTANCE_ID"
	EnvNameNatsURL               = "NATS_URL"

	ContainerNameInfoServer = "info-server"
	ContainerNameGateway    = "gateway"

	BucketNameFunctions = "fid-functions"
	BucketNameElections = "fid-elections"
	BucketNameInstances = "fid-instances"

	FilenameFidfile = "Fidfile.yaml"

	PortTCP80 = "80/tcp"
)
