//go:build !arm && !arm64

package glplatform

type GLContext interface {
	Init() error
}

var (
	InternalFormatRGBA int32
	VersionMajor       int
	VersionMinor       int
	ProfileMask        uint32
	ShaderVersion      string
)

func Init() error {
	return platformInit()
}

func platformInit() error
