package d2systems

import (
	"github.com/gravestench/akara"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2sprite"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2components"
)

const (
	fmtCreateSpriteErr = "could not create sprite from image `%s` and palette `%s`"
)

// NewSpriteFactorySubsystem creates a new sprite factory which is intended
// to be embedded in the game object factory system.
func NewSpriteFactorySubsystem(b akara.BaseSystem, l *d2util.Logger) *SpriteFactory {
	sys := &SpriteFactory{
		Logger: l,
	}

	sys.BaseSystem = b

	sys.World.AddSystem(sys)

	return sys
}

type spriteLoadQueueEntry struct {
	spriteImage, spritePalette akara.EID
}

type spriteLoadQueue = map[akara.EID]spriteLoadQueueEntry

// SpriteFactory is responsible for queueing sprites to be loaded (as spriteations),
// as well as binding the spriteation to a renderer if one is present (which generates the sprite surfaces).
type SpriteFactory struct {
	akara.BaseSubscriberSystem
	*d2util.Logger
	RenderSystem *RenderSystem
	d2components.FilePathFactory
	d2components.PositionFactory
	d2components.Dc6Factory
	d2components.DccFactory
	d2components.PaletteFactory
	d2components.SpriteFactory
	d2components.TextureFactory
	d2components.OriginFactory
	d2components.SegmentedSpriteFactory
	loadQueue       spriteLoadQueue
	spritesToRender *akara.Subscription
	spritesToUpdate *akara.Subscription
}

// Init the sprite factory, injecting the necessary components
func (t *SpriteFactory) Init(world *akara.World) {
	t.World = world

	t.Info("initializing sprite factory ...")

	t.setupFactories()
	t.setupSubscriptions()

	t.loadQueue = make(spriteLoadQueue)
}

func (t *SpriteFactory) setupFactories() {
	t.InjectComponent(&d2components.FilePath{}, &t.FilePath)
	t.InjectComponent(&d2components.Position{}, &t.Position)
	t.InjectComponent(&d2components.Dc6{}, &t.Dc6)
	t.InjectComponent(&d2components.Dcc{}, &t.Dcc)
	t.InjectComponent(&d2components.Palette{}, &t.Palette)
	t.InjectComponent(&d2components.Texture{}, &t.Texture)
	t.InjectComponent(&d2components.Origin{}, &t.Origin)
	t.InjectComponent(&d2components.Sprite{}, &t.SpriteFactory.Sprite)
	t.InjectComponent(&d2components.SegmentedSprite{}, &t.SegmentedSpriteFactory.SegmentedSprite)
}

func (t *SpriteFactory) setupSubscriptions() {
	spritesToRender := t.NewComponentFilter().
		Require(&d2components.Sprite{}). // we want to process entities that have an spriteation ...
		Forbid(&d2components.Texture{}). // ... but are missing a surface
		Build()

	spritesToUpdate := t.NewComponentFilter().
		Require(&d2components.Sprite{}).  // we want to process entities that have an spriteation ...
		Require(&d2components.Texture{}). // ... but are missing a surface
		Build()

	t.spritesToRender = t.AddSubscription(spritesToRender)
	t.spritesToUpdate = t.AddSubscription(spritesToUpdate)
}

// Update processes the load queue which attempting to create spriteations, as well as
// binding existing spriteations to a renderer if one is present.
func (t *SpriteFactory) Update() {
	for spriteID := range t.loadQueue {
		t.tryCreatingSprite(spriteID)
	}

	for _, eid := range t.spritesToUpdate.GetEntities() {
		t.updateSprite(eid)
	}

	for _, eid := range t.spritesToRender.GetEntities() {
		t.tryRenderingSprite(eid)
	}
}

// Sprite queues a sprite spriteation to be loaded
func (t *SpriteFactory) Sprite(x, y float64, imgPath, palPath string) akara.EID {
	spriteID := t.NewEntity()

	position := t.AddPosition(spriteID)
	position.X, position.Y = x, y

	imgID, palID := t.NewEntity(), t.NewEntity()
	t.AddFilePath(imgID).Path = imgPath
	t.AddFilePath(palID).Path = palPath

	t.loadQueue[spriteID] = spriteLoadQueueEntry{
		spriteImage:   imgID,
		spritePalette: palID,
	}

	return spriteID
}

// SegmentedSprite queues a segmented sprite spriteation to be loaded.
// A segmented sprite is a sprite that has many frames that form the entire sprite.
func (t *SpriteFactory) SegmentedSprite(x, y float64, imgPath, palPath string, xseg, yseg, frame int) akara.EID {
	spriteID := t.Sprite(x, y, imgPath, palPath)

	s := t.AddSegmentedSprite(spriteID)
	s.Xsegments = xseg
	s.Ysegments = yseg
	s.FrameOffset = frame

	return spriteID
}

func (t *SpriteFactory) tryCreatingSprite(id akara.EID) {
	entry := t.loadQueue[id]
	imageID, paletteID := entry.spriteImage, entry.spritePalette

	imagePath, found := t.GetFilePath(imageID)
	if !found {
		return
	}

	palettePath, found := t.GetFilePath(paletteID)
	if !found {
		return
	}

	palette, found := t.GetPalette(paletteID)
	if !found {
		return
	}

	var sprite d2interface.Sprite

	var err error

	if dc6, found := t.GetDc6(imageID); found {
		sprite, err = t.createDc6Sprite(dc6, palette)
	}

	if dcc, found := t.GetDcc(imageID); found {
		sprite, err = t.createDccSprite(dcc, palette)
	}

	if err != nil {
		t.Errorf(fmtCreateSpriteErr, imagePath.Path, palettePath.Path)

		t.RemoveEntity(id)
		t.RemoveEntity(imageID)
		t.RemoveEntity(paletteID)
	}

	t.AddSprite(id).Sprite = sprite

	delete(t.loadQueue, id)
}

func (t *SpriteFactory) tryRenderingSprite(eid akara.EID) {
	if t.RenderSystem == nil {
		return
	}

	if t.RenderSystem.renderer == nil {
		return
	}

	sprite, found := t.GetSprite(eid)
	if !found {
		return
	}

	if sprite.Sprite == nil {
		return
	}

	sprite.BindRenderer(t.RenderSystem.renderer)

	sfc := sprite.GetCurrentFrameSurface()

	t.AddTexture(eid).Texture = sfc
}

func (t *SpriteFactory) updateSprite(eid akara.EID) {
	if t.RenderSystem == nil {
		return
	}

	if t.RenderSystem.renderer == nil {
		return
	}

	sprite, found := t.GetSprite(eid)
	if !found {
		return
	}

	if sprite.Sprite == nil {
		return
	}

	texture, found := t.GetTexture(eid)
	if !found {
		return
	}

	origin, found := t.GetOrigin(eid)
	if !found {
		origin = t.AddOrigin(eid)
	}

	_ = sprite.Sprite.Advance(t.World.TimeDelta)

	texture.Texture = sprite.GetCurrentFrameSurface()

	ox, oy := sprite.GetCurrentFrameOffset()
	origin.X, origin.Y = float64(ox), float64(oy)

	if _, isSegmented := t.GetSegmentedSprite(eid); !isSegmented {
		_, frameHeight := sprite.GetCurrentFrameSize()
		origin.Y -= float64(frameHeight)
	}
}

func (t *SpriteFactory) createDc6Sprite(
	dc6 *d2components.Dc6,
	pal *d2components.Palette,
) (d2interface.Sprite, error) {
	return d2sprite.NewDC6Sprite(dc6.DC6, pal.Palette, 0)
}

func (t *SpriteFactory) createDccSprite(
	dcc *d2components.Dcc,
	pal *d2components.Palette,
) (d2interface.Sprite, error) {
	return d2sprite.NewDCCSprite(dcc.DCC, pal.Palette, 0)
}