package docker

import (
	"time"

	"github.com/zhulik/fid/internal/core"
)

type Function struct {
	Name_    string            `json:"name"`
	Image_   string            `json:"image"`
	Timeout_ time.Duration     `json:"timeout"`
	MinScale int64             `json:"minScale"`
	MaxScale int64             `json:"maxScale"`
	Env_     map[string]string `json:"env"`
}

func (f Function) Image() string {
	return f.Image_
}

func (f Function) Env() map[string]string {
	return f.Env_
}

func (f Function) Name() string {
	return f.Name_
}

func (f Function) String() string {
	return f.Name_
}

func (f Function) Timeout() time.Duration {
	return f.Timeout_
}

func (f Function) ScalingConfig() core.ScalingConfig {
	return core.ScalingConfig{
		Min: f.MinScale,
		Max: f.MaxScale,
	}
}
