package graphics

import (
	"fmt"
	"log"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
)

func CreateTextureFromSurface(surface *sdl.Surface) (uint32, error) {

	if surface == nil {
		return 0, fmt.Errorf("CreateTextureFromSurface: input surface is nil")
	}

	var textureID uint32
	var err error

	surfaceToUpload := surface
	var convertedSurface *sdl.Surface = nil

	sdlFormatEnum := surface.Format.Format
	var dataFormat uint32
	var internalFormat int32 = gl.RGBA8

	if sdlFormatEnum == sdl.PIXELFORMAT_INDEX8 {
		convertedSurface, err = surface.ConvertFormat(sdl.PIXELFORMAT_ARGB8888, 0)
		if err != nil {
			return 0, fmt.Errorf("failed to convert INDEX8 surface to ARGB8888: %w", err)
		}
		defer convertedSurface.Free()

		surfaceToUpload = convertedSurface
		sdlFormatEnum = surfaceToUpload.Format.Format
	}

	switch sdlFormatEnum {
	case sdl.PIXELFORMAT_ARGB8888:
		dataFormat = gl.BGRA
	case sdl.PIXELFORMAT_ABGR8888:
		dataFormat = gl.RGBA
	case sdl.PIXELFORMAT_RGBA8888:
		dataFormat = gl.RGBA
	case sdl.PIXELFORMAT_BGRA8888:
		dataFormat = gl.BGRA
	default:
		return 0, fmt.Errorf("unhandled SDL pixel format for OpenGL upload: %s", sdl.GetPixelFormatName(uint(sdlFormatEnum)))
	}

	for errCode := gl.GetError(); errCode != gl.NO_ERROR; errCode = gl.GetError() {
		log.Printf("OpenGL error pending *inside* CreateTextureFromSurface before GenTextures: 0x%X", errCode)
	}
	gl.GenTextures(1, &textureID) // The crash happens here. Everything looks... Good?
	if errCode := gl.GetError(); errCode != gl.NO_ERROR {
		return 0, fmt.Errorf("OpenGL error 0x%X after GenTextures", errCode)
	}
	if textureID == 0 {
		return 0, fmt.Errorf("gl.GenTextures returned textureID 0")
	}

	gl.BindTexture(gl.TEXTURE_2D, textureID)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)

	expectedPitch := surfaceToUpload.W * int32(surfaceToUpload.Format.BytesPerPixel)
	if int32(surfaceToUpload.Pitch) != expectedPitch {
		gl.PixelStorei(gl.UNPACK_ROW_LENGTH, surfaceToUpload.Pitch/int32(surfaceToUpload.Format.BytesPerPixel))
	} else {
		gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
	}

	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		internalFormat,
		surfaceToUpload.W,
		surfaceToUpload.H,
		0,
		dataFormat,
		gl.UNSIGNED_BYTE,
		gl.Ptr(surfaceToUpload.Pixels()),
	)

	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)

	if glErr := gl.GetError(); glErr != gl.NO_ERROR {
		gl.DeleteTextures(1, &textureID)
		return 0, fmt.Errorf("OpenGL error 0x%X after TexImage2D", glErr)
	}

	gl.BindTexture(gl.TEXTURE_2D, 0)
	return textureID, nil
}

// CheckGLError queries OpenGL for errors and prints them to the console.
// Call this after a block of GL calls you want to debug.
func CheckGLError() error {
	for {
		errCode := gl.GetError()
		if errCode == gl.NO_ERROR {
			break // No more errors
		}

		var errString string
		switch errCode {
		case gl.INVALID_ENUM:
			errString = "INVALID_ENUM"
		case gl.INVALID_VALUE:
			errString = "INVALID_VALUE"
		case gl.INVALID_OPERATION:
			errString = "INVALID_OPERATION"
		case gl.STACK_OVERFLOW:
			errString = "STACK_OVERFLOW"
		case gl.STACK_UNDERFLOW:
			errString = "STACK_UNDERFLOW"
		case gl.OUT_OF_MEMORY:
			errString = "OUT_OF_MEMORY"
		case gl.INVALID_FRAMEBUFFER_OPERATION:
			errString = "INVALID_FRAMEBUFFER_OPERATION"
		default:
			errString = fmt.Sprintf("Unknown error code: %d", errCode)
		}

		return fmt.Errorf(errString)
	}
	return nil
}

// DEPRECATED: SetDisplayOrientation is a placeholder for compatibility.
// To be removed
func SetDisplayOrientation(w int32, h int32) (int32, int32) {
	return w, h
}
