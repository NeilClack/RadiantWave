package shaderManager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-gl/gl/v3.3-core/gl"
	"radiantwavetech.com/radiantwave/internal/db"
	"radiantwavetech.com/radiantwave/internal/logger"
)

// Shader represents a compiled GLSL program and its associated data.
type Shader struct {
	Name      string
	ProgramID uint32
	Render    func() // This will be used in the future.
}

// ShaderManager handles the loading, compiling, and management of GLSL shader programs.
type ShaderManager struct {
	shaders map[string]*Shader
}

var (
	instance *ShaderManager
	once     sync.Once
)

// InitShaderManager initializes the ShaderManager singleton.
func InitShaderManager() error {
	var err error
	once.Do(func() {
		logger.InfoF("Initializing ShaderManager singleton.")
		sm := &ShaderManager{
			shaders: make(map[string]*Shader),
		}
		if e := sm.LoadShaders(); e != nil {
			err = fmt.Errorf("could not load shaders: %w", e)
			return
		}
		instance = sm
		logger.InfoF("ShaderManager initialized.")
	})
	return err
}

// Get returns the singleton ShaderManager instance.
func Get() *ShaderManager {
	if instance == nil {
		panic("ShaderManager has not been initialized. Call InitShaderManager at application startup.")
	}
	return instance
}

// Get retrieves a compiled Shader object by its name.
func (sm *ShaderManager) Get(name string) (*Shader, bool) {
	shader, ok := sm.shaders[name]
	return shader, ok
}

// ListShaders returns all shaders.
func (sm *ShaderManager) ListShaders() map[string]*Shader {
	return sm.shaders
}

// Close deletes all loaded shader programs from the GPU to free up resources.
func (sm *ShaderManager) Close() {
	for name, shader := range sm.shaders {
		gl.DeleteProgram(shader.ProgramID)
		delete(sm.shaders, name)
	}
	logger.InfoF("ShaderManager closed and all programs deleted.")
}

// LoadShaders scans a directory for .vert and .frag files and creates a shader program for each pair.
func (sm *ShaderManager) LoadShaders() error {
	// Get assets directory from database
	assetsDir, err := db.GetConfigValue("assets_dir")
	if err != nil {
		return fmt.Errorf("retrieving assets_dir from db: %w", err)
	}

	srcDir := filepath.Join(assetsDir, "shaders")
	shaderFiles := make(map[string]string)

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".vert") {
			name := strings.TrimSuffix(info.Name(), ".vert")
			shaderFiles[name] = path
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error walking shader directory: %w", err)
	}

	for name, vertPath := range shaderFiles {
		fragPath := strings.Replace(vertPath, ".vert", ".frag", 1)
		if _, err := os.Stat(fragPath); os.IsNotExist(err) {
			logger.WarningF("Vertex shader '%s' found but matching fragment shader '%s' is missing. Skipping.", vertPath, fragPath)
			continue
		}

		programID, err := createShaderProgram(vertPath, fragPath)
		if err != nil {
			logger.ErrorF("Failed to create shader program for '%s': %v", name, err)
			continue
		}

		newShader := &Shader{
			Name:      name,
			ProgramID: programID,
			Render:    nil, // Render function is not yet assigned.
		}
		sm.shaders[name] = newShader
		logger.InfoF("Successfully compiled and linked shader: %s", name)
	}

	return nil
}

// createShaderProgram reads, compiles, and links a vertex and fragment shader from file paths.
func createShaderProgram(vertexPath, fragmentPath string) (uint32, error) {
	vertexSource, err := os.ReadFile(vertexPath)
	if err != nil {
		return 0, fmt.Errorf("reading vertex shader %s: %w", vertexPath, err)
	}
	fragmentSource, err := os.ReadFile(fragmentPath)
	if err != nil {
		return 0, fmt.Errorf("reading fragment shader %s: %w", fragmentPath, err)
	}

	vertexShader, err := compileShader(string(vertexSource)+"\x00", gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}
	defer gl.DeleteShader(vertexShader)

	fragmentShader, err := compileShader(string(fragmentSource)+"\x00", gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}
	defer gl.DeleteShader(fragmentShader)

	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))
		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	return program, nil
}

// compileShader is a helper function that compiles a single shader source string.
func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		var shaderTypeName string
		switch shaderType {
		case gl.VERTEX_SHADER:
			shaderTypeName = "Vertex"
		case gl.FRAGMENT_SHADER:
			shaderTypeName = "Fragment"
		default:
			shaderTypeName = "Unknown"
		}
		return 0, fmt.Errorf("failed to compile %s shader: %v", shaderTypeName, log)
	}

	return shader, nil
}
