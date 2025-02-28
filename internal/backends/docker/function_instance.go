package docker

import (
	"time"

	"github.com/zhulik/fid/internal/core"
)

type FunctionInstance struct {
	id           string
	lastExecuted time.Time
	function     core.FunctionDefinition
}

func NewFunctionInstance(entry core.KVEntry, definition core.FunctionDefinition) FunctionInstance {
	_, id := parseKey(entry.Key)

	return FunctionInstance{
		id:           id,
		lastExecuted: deserializeTime(entry.Value),
		function:     definition,
	}
}

func (f FunctionInstance) ID() string {
	return f.id
}

func (f FunctionInstance) LastExecuted() time.Time {
	return f.lastExecuted
}

func (f FunctionInstance) Function() core.FunctionDefinition {
	return f.function
}
