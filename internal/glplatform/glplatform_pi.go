//go:build arm || arm64

package glplatform

import (
	"github.com/go-gl/gl/v3.1/gles2"
	"github.com/veandco/go-sdl2/sdl"
)

func init() {
	InternalFormatRGBA = gles2.RGBA
	VersionMajor = 2
	VersionMinor = 0
	ProfileMask = sdl.GL_CONTEXT_PROFILE_ES
	ShaderVersion = "#version 100"
}

func platformInit() error {
	return gles2.Init()
}

// Re-export gles2 package for use
var GL = gles2
