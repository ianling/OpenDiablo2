package d2systems

import (
	"time"

	"github.com/gravestench/ecs"
)

const (
	defaultScale float64 = 1
)

// NewTimeScaleSystem creates a timescale system
func NewTimeScaleSystem() *TimeScaleSystem {
	m := &TimeScaleSystem{
		BaseSystem: &ecs.BaseSystem{},
	}

	return m
}

// static check that TimeScaleSystem implements the System interface
var _ ecs.System = &TimeScaleSystem{}

// TimeScaleSystem should be the first system added to the world, and whose only job is to
// apply a scalar the world's TimeDelta between frames. It's useful for slowing down or speeding
// up the game time without affecting the render rate.
type TimeScaleSystem struct {
	*ecs.BaseSystem
	scale float64
}

// Init will initialize the TimeScale system
func (t *TimeScaleSystem) Init(world *ecs.World) {
	t.World = world
	t.scale = defaultScale
}

// Process scales the worlds time delta for this frame
func (t *TimeScaleSystem) Process() {
	if !t.Active() {
		return
	}

	t.World.TimeDelta *= time.Duration(t.scale)
}
