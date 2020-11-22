package d2systems

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/gravestench/akara"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2components"
)

func TestMovementSystem_Init(t *testing.T) {
	cfg := akara.NewWorldConfig()

	cfg.With(NewMovementSystem())

	world := akara.NewWorld(cfg)

	if len(world.Systems) != 1 {
		t.Error("system not added to the world")
	}
}

func TestMovementSystem_Active(t *testing.T) {
	sys := NewMovementSystem()

	if sys.Active() {
		t.Error("system should not be active at creation")
	}
}

func TestMovementSystem_SetActive(t *testing.T) {
	sys := NewMovementSystem()

	sys.SetActive(false)

	if sys.Active() {
		t.Error("system should be inactive after being set inactive")
	}
}

func TestMovementSystem_EntityAdded(t *testing.T) {
	cfg := akara.NewWorldConfig()

	sys := NewMovementSystem()

	cfg.With(sys).
		With(d2components.NewPositionMap()).
		With(d2components.NewVelocityMap())

	world := akara.NewWorld(cfg)

	e := world.NewEntity()

	position := sys.AddPosition(e)
	velocity := sys.AddVelocity(e)

	px, py := 10., 10.
	vx, vy := 1., 0.

	position.Set(px, py)
	velocity.Set(vx, vy)

	if len(sys.Subscriptions[0].GetEntities()) != 1 {
		t.Error("entity not added to the system")
	}

	if p, found := sys.GetPosition(e); !found {
		t.Error("position component not found")
	} else if p.X() != px || p.Y() != py {
		fmtError := "position component values incorrect:\n\t expected %v, %v but got %v, %v"
		t.Errorf(fmtError, px, py, p.X(), p.Y())
	}

	if v, found := sys.GetVelocity(e); !found {
		t.Error("position component not found")
	} else if v.X() != vx || v.Y() != vy {
		fmtError := "velocity component values incorrect:\n\t expected %v, %v but got %v, %v"
		t.Errorf(fmtError, px, py, v.X(), v.Y())
	}
}

func TestMovementSystem_Update(t *testing.T) {
	// world configFileBootstrap
	cfg := akara.NewWorldConfig()

	movementSystem := NewMovementSystem()
	positions := d2components.NewPositionMap()
	velocities := d2components.NewVelocityMap()

	cfg.With(movementSystem).With(positions).With(velocities)

	world := akara.NewWorld(cfg)

	// lets make an entity and add some components to it
	e := world.NewEntity()
	position := movementSystem.AddPosition(e)
	velocity := movementSystem.AddVelocity(e)

	px, py := 10., 10.
	vx, vy := 1., -1.

	// mutate the components a bit
	position.Set(px, py)
	velocity.Set(vx, vy)

	// should apply the velocity to the position
	_ = world.Update(time.Second)

	if position.X() != px+vx || position.Y() != py+vy {
		fmtError := "expected position (%v, %v) but got (%v, %v)"
		t.Errorf(fmtError, px+vx, py+vy, position.X(), position.Y())
	}
}

func benchN(n int, b *testing.B) {
	cfg := akara.NewWorldConfig()

	movementSystem := NewMovementSystem()

	cfg.With(movementSystem)

	world := akara.NewWorld(cfg)

	for idx := 0; idx < n; idx++ {
		e := world.NewEntity()
		p := movementSystem.AddPosition(e)
		v := movementSystem.AddVelocity(e)

		p.Set(0, 0)
		v.Set(rand.Float64(), rand.Float64()) //nolint:gosec // it's just a test
	}

	benchName := strconv.Itoa(n) + "_entity update"
	b.Run(benchName, func(b *testing.B) {
		for idx := 0; idx < b.N; idx++ {
			_ = world.Update(time.Millisecond)
		}
	})
}

func BenchmarkMovementSystem_Update(b *testing.B) {
	benchN(1e1, b)
	benchN(1e2, b)
	benchN(1e3, b)
	benchN(1e4, b)
	benchN(1e5, b)
	benchN(1e6, b)
}