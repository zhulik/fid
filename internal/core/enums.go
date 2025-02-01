package core

type SubjectName = string

const (
	// InvokeSubjectBase used as fid.invoke.<function_name>.
	InvokeSubjectBase SubjectName = "fid.invoke"

	// ResponseSubjectBase used as fid.response.<function_name>.<request_id>.response or fid.response.<request_id>.error.
	ResponseSubjectBase SubjectName = "fid.response"
)
