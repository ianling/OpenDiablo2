package d2systems

import (
	"image/color"
	"math"

	"github.com/gravestench/akara"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2components"
)

const (
	sceneKeyLoading = "Loading"
)

// static check that LoadingScene implements the scene interface
var _ d2interface.Scene = &LoadingScene{}

// NewLoadingScene creates a new main menu scene. This is the first screen that the user
// will see when launching the game.
func NewLoadingScene() *LoadingScene {
	scene := &LoadingScene{
		BaseScene: NewBaseScene(sceneKeyLoading),
	}

	return scene
}

// LoadingScene represents the game's loading screen, where loading progress is displayed
type LoadingScene struct {
	*BaseScene
	loadingSprite akara.EID
	loadStages    struct {
		stage1 *akara.Subscription // has path, no type
		stage2 *akara.Subscription // has type, no handle
		stage3 *akara.Subscription // has handle, no asset
		stage4 *akara.Subscription // is loaded
	}
	progress float64
	booted   bool
}

// Init the loading scene
func (s *LoadingScene) Init(world *akara.World) {
	s.World = world

	s.Info("initializing ...")

	s.backgroundColor = color.Black

	s.setupSubscriptions()
}

func (s *LoadingScene) setupSubscriptions() {
	s.Info("setting up component subscriptions")

	stage1 := s.NewComponentFilter().
		Require(
			&d2components.FilePath{},
		).
		Forbid( // but we forbid files that are already loaded
			&d2components.FileType{},
			&d2components.FileHandle{},
			&d2components.FileSource{},
			&d2components.GameConfig{},
			&d2components.StringTable{},
			&d2components.DataDictionary{},
			&d2components.Palette{},
			&d2components.PaletteTransform{},
			&d2components.Cof{},
			&d2components.Dc6{},
			&d2components.Dcc{},
			&d2components.Ds1{},
			&d2components.Dt1{},
			&d2components.Wav{},
			&d2components.AnimationData{},
		).
		Build()

	stage2 := s.NewComponentFilter().
		Require(
			&d2components.FilePath{},
			&d2components.FileType{},
		).
		Forbid( // but we forbid files that are already loaded
			&d2components.FileHandle{},
			&d2components.FileSource{},
			&d2components.GameConfig{},
			&d2components.StringTable{},
			&d2components.DataDictionary{},
			&d2components.Palette{},
			&d2components.PaletteTransform{},
			&d2components.Cof{},
			&d2components.Dc6{},
			&d2components.Dcc{},
			&d2components.Ds1{},
			&d2components.Dt1{},
			&d2components.Wav{},
			&d2components.AnimationData{},
		).
		Build()

	stage3 := s.NewComponentFilter().
		Require(
			&d2components.FilePath{},
			&d2components.FileType{},
			&d2components.FileHandle{},
		).
		Forbid( // but we forbid files that are already loaded
			&d2components.FileSource{},
			&d2components.GameConfig{},
			&d2components.StringTable{},
			&d2components.DataDictionary{},
			&d2components.Palette{},
			&d2components.PaletteTransform{},
			&d2components.Cof{},
			&d2components.Dc6{},
			&d2components.Dcc{},
			&d2components.Ds1{},
			&d2components.Dt1{},
			&d2components.Wav{},
			&d2components.AnimationData{},
		).
		Build()

	// we want to know about loaded files, too
	stage4 := s.NewComponentFilter().
		RequireOne(
			&d2components.FileHandle{},
			&d2components.FileSource{},
			&d2components.GameConfig{},
			&d2components.StringTable{},
			&d2components.DataDictionary{},
			&d2components.Palette{},
			&d2components.PaletteTransform{},
			&d2components.Cof{},
			&d2components.Dc6{},
			&d2components.Dcc{},
			&d2components.Ds1{},
			&d2components.Dt1{},
			&d2components.Wav{},
			&d2components.AnimationData{},
		).
		Build()

	s.loadStages.stage1 = s.World.AddSubscription(stage1) // has path, no type
	s.loadStages.stage2 = s.World.AddSubscription(stage2) // has type, no handle
	s.loadStages.stage3 = s.World.AddSubscription(stage3) // has handle, no asset
	s.loadStages.stage4 = s.World.AddSubscription(stage4) // is loaded
}

func (s *LoadingScene) boot() {
	if !s.BaseScene.booted {
		s.BaseScene.boot()
		return
	}

	s.createLoadingScreen()

	s.booted = true
}

func (s *LoadingScene) createLoadingScreen() {
	s.Info("creating loading screen")
	s.loadingSprite = s.Add.Sprite(0, 0, d2resource.LoadingScreen, d2resource.PaletteLoading)
}

// Update the loading scene
func (s *LoadingScene) Update() {
	for _, id := range s.Viewports {
		s.AddPriority(id).Priority = scenePriorityLoading
	}

	if s.Paused() {
		return
	}

	if !s.booted {
		s.boot()
	}

	s.updateLoadProgress()
	s.updateViewportAlpha()
	s.updateLoadingSpritePosition()
	s.updateLoadingSpriteFrame()

	s.BaseScene.Update()
}

func (s *LoadingScene) updateLoadProgress() {
	untyped := float64(len(s.loadStages.stage1.GetEntities()))
	unhandled := float64(len(s.loadStages.stage2.GetEntities()))
	unparsed := float64(len(s.loadStages.stage3.GetEntities()))
	loaded := float64(len(s.loadStages.stage4.GetEntities()))

	s.progress = 1 - ((untyped + unhandled + unparsed) / 3 / loaded)
}

func (s *LoadingScene) updateViewportAlpha() {
	if len(s.Viewports) < 1 {
		return
	}

	alpha, found := s.GetAlpha(s.Viewports[0])
	if !found {
		return
	}

	isLoading := len(s.loadStages.stage1.GetEntities()) > 0 ||
		len(s.loadStages.stage2.GetEntities()) > 0 ||
		len(s.loadStages.stage3.GetEntities()) > 0

	if isLoading {
		alpha.Alpha = math.Min(alpha.Alpha+0.125, 1)
	} else {
		alpha.Alpha = math.Max(alpha.Alpha-0.125, 0)
	}
}

func (s *LoadingScene) updateLoadingSpritePosition() {
	if len(s.Viewports) < 1 {
		return
	}

	viewport, found := s.GetViewport(s.Viewports[0])
	if !found {
		return
	}

	sprite, found := s.GetSprite(s.loadingSprite)
	if !found {
		return
	}

	position, found := s.GetPosition(s.loadingSprite)
	if !found {
		return
	}

	centerX, centerY := viewport.Width/2, viewport.Height/2
	frameW, frameH := sprite.GetCurrentFrameSize()

	// we add the frameH in the Y because sprites are supposed to be drawn from bottom to top
	position.X, position.Y = float64(centerX-(frameW/2)), float64(centerY+(frameH/2))
}

func (s *LoadingScene) updateLoadingSpriteFrame() {
	sprite, found := s.GetSprite(s.loadingSprite)
	if !found {
		return
	}

	numFrames := float64(sprite.GetFrameCount())
	if err := sprite.SetCurrentFrame(int(s.progress * (numFrames - 1))); err != nil {
		_ = sprite.SetCurrentFrame(0)
	}
}