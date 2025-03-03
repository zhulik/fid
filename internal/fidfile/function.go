package fidfile

import (
	"time"

	"github.com/zhulik/fid/internal/core"
)

type Function struct {
	Name_    string            `validate:"required"           yaml:"-"`
	Image_   string            `validate:"required"           yaml:"image"`
	Env_     map[string]string `yaml:"env"`
	Min      int               `validate:"gte=0,ltefield=Max" yaml:"min"`
	Max      int               `validate:"gte=0,gtefield=Min" yaml:"max"`
	Timeout_ time.Duration     `validate:"required,gte=1s"    yaml:"timeout"`
}

func (f Function) Name() string {
	return f.Name_
}

func (f Function) String() string {
	return f.Name_
}

func (f Function) Image() string {
	return f.Image_
}

func (f Function) Timeout() time.Duration {
	return f.Timeout_
}

func (f Function) ScalingConfig() core.ScalingConfig {
	return core.ScalingConfig{
		Min: f.Min,
		Max: f.Max,
	}
}

func (f Function) Env() map[string]string {
	return f.Env_
}
