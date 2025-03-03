package docker

import (
	"time"

	"github.com/zhulik/fid/internal/core"
)

type FunctionInstance struct {
	ID_           string
	LastExecuted_ time.Time
	Busy_         bool
	Function_     core.FunctionDefinition
}

func NewFunctionInstance(id string, function core.FunctionDefinition, values map[string]core.KVEntry) FunctionInstance {
	instance := FunctionInstance{
		ID_:       id,
		Function_: function,
	}

	// if lastExecuted record exist - parse it and assign
	if entry, ok := values[lastExecutedKey(function.Name(), id)]; ok {
		instance.LastExecuted_ = deserializeTime(entry.Value)
	}

	// If no idle flag - mark as busy
	if _, ok := values[idleKey(function.Name(), id)]; !ok {
		instance.Busy_ = true
	}

	return instance
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

func (f FunctionInstance) Busy() bool {
	return f.Busy_
}
