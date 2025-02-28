package docker

import (
	"time"

	"github.com/zhulik/fid/internal/core"
)

type FunctionInstance struct {
	ID_           string
	LastExecuted_ time.Time
	Function_     core.FunctionDefinition
}

func NewFunctionInstance(entry core.KVEntry, definition core.FunctionDefinition) FunctionInstance {
	_, id := parseKey(entry.Key)

	return FunctionInstance{
		ID_:           id,
		LastExecuted_: deserializeTime(entry.Value),
		Function_:     definition,
	}
}

func (f FunctionInstance) ID() string {
	return f.ID_
}

func (f FunctionInstance) LastExecuted() time.Time {
	return f.LastExecuted_
}

func (f FunctionInstance) Function() core.FunctionDefinition {
	return f.Function_
}
