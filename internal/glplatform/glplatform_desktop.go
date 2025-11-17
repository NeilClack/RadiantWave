package glplatform

import (
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
)

func init() {
	InternalFormatRGBA = gl.RGBA8
	VersionMajor = 3
	VersionMinor = 3
	ProfileMask = sdl.GL_CONTEXT_PROFILE_CORE
	ShaderVersion = "#version 330 core\n"
}

func platormInit() error {
	return gl.Init()
}

var GL = gl
