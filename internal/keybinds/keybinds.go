// package keybinds provides keybind functionality for the entire application
package keybinds

import (
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiant_wave/internal/logger"
)

// KeyCombo represents a combination of a key and it's active modifier.
type KeyCombo struct {
	Key sdl.Keycode
	Mod sdl.Keymod
}

// Binds contains all currently registered Keybinds.
var Binds = make(map[KeyCombo]func())

// Regiser registers a new keybind in Binds
func Register(k sdl.Keycode, m sdl.Keymod, action func()) {
	mod := NormalizeModifiers(m)
	newCombo := KeyCombo{k, mod}
	_, ok := Binds[newCombo]
	if ok {
		logger.LogError("Unable to bind key combo, combo already exists")
	}
	Binds[newCombo] = action
}

// BindKey adds a new KeyCombo and Function (the action to take) to the keybinds.Binds
func BindKey(k KeyCombo, f func()) {
	k.Mod = NormalizeModifiers(k.Mod)
	Binds[k] = f
}

// UnbindKey removes a KeyCombo from the available binds
func UnbindKey(k KeyCombo) {
	k.Mod = NormalizeModifiers(k.Mod)
	delete(Binds, k)
}

func PerformAction(event sdl.Event) {
	switch e := event.(type) {
	case *sdl.KeyboardEvent:
		if e.Type == sdl.KEYDOWN {
			key := e.Keysym.Sym
			rawMod := e.Keysym.Mod
			mod := NormalizeModifiers(sdl.Keymod(rawMod))
			combo := KeyCombo{key, mod}
			action, ok := Binds[combo]
			if ok {
				action()
			}
		}
	}
}

// NormalizeModifiers converts a raw SDL modifier bitmask into a canonical form
// focusing on common modifiers (SHIFT, CTRL, ALT, GUI) and ignoring others like CapsLock.
func NormalizeModifiers(mod sdl.Keymod) sdl.Keymod {
	var normalized sdl.Keymod = sdl.KMOD_NONE
	if mod&sdl.KMOD_SHIFT != 0 {
		normalized |= sdl.KMOD_SHIFT
	}
	if mod&sdl.KMOD_CTRL != 0 {
		normalized |= sdl.KMOD_CTRL
	}
	if mod&sdl.KMOD_ALT != 0 {
		normalized |= sdl.KMOD_ALT
	}
	if mod&sdl.KMOD_GUI != 0 {
		normalized |= sdl.KMOD_GUI
	}
	return normalized
}
