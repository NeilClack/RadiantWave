// Package page provides the interface for pages
package page

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/veandco/go-sdl2/sdl"
	"radiantwavetech.com/radiantwave/internal/graphics"
	"radiantwavetech.com/radiantwave/internal/logger"
	"radiantwavetech.com/radiantwave/internal/shaderManager"
)

// Page defines the contract for all application screens.
type Page interface {
	Init(app ApplicationInterface) error
	HandleEvent(event *sdl.Event) error
	Update(dt float32) error
	Render() error
	// Destroy cleans up the page's resources. The Application's page manager
	// is responsible for calling this method
	Destroy() error
}

// ApplicationInterface defines the set of methods that a page can use to
// interact with the main application.
type ApplicationInterface interface {
	SwitchPage(p Page)
	PushPage(p Page)
	UnwindToPage(p Page)
	PopPage()
	GetDrawableSize() (int32, int32)
}

// Base is a helper struct that pages can embed to get common functionality.
type Base struct {
	App           ApplicationInterface
	QuadVAO       uint32
	QuadVBO       uint32
	Projection    mgl32.Mat4
	ScreenWidth   int32
	ScreenHeight  int32
	ScreenCenterX int32
	ScreenCenterY int32
}

// Init stores the application context and sets up common resources.
func (p *Base) Init(app ApplicationInterface) error {
	p.App = app

	vertices := []float32{
		// pos(x,y) tex(u,v)
		0.0, 1.0, 0.0, 1.0,
		1.0, 0.0, 1.0, 0.0,
		0.0, 0.0, 0.0, 0.0,

		0.0, 1.0, 0.0, 1.0,
		1.0, 1.0, 1.0, 1.0,
		1.0, 0.0, 1.0, 0.0,
	}

	screenWidth, screenHeight := p.App.GetDrawableSize()
	p.Projection = mgl32.Ortho(0, float32(screenWidth), 0, float32(screenHeight), -1, 1)
	p.ScreenHeight = screenHeight
	p.ScreenWidth = screenWidth
	p.ScreenCenterX = screenWidth / 2
	p.ScreenCenterY = screenHeight / 2

	gl.GenVertexArrays(1, &p.QuadVAO)
	gl.GenBuffers(1, &p.QuadVBO)
	gl.BindVertexArray(p.QuadVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, p.QuadVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	stride := int32(4 * 4) // Each vertex has 4 floats (pos.x, pos.y, tex.u, tex.v)

	// Attribute 0: Vertex Position (vec2)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, stride, 0)

	// Attribute 1: Texture Coordinate (vec2)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, stride, 2*4) // Offset by 2 floats

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)
	gl.Disable(gl.BLEND)
	gl.UseProgram(0)
	return nil
}

// Destroy cleans up the common OpenGL resources.
func (p *Base) Destroy() error {
	logger.InfoF("Destroying base resources: (VAO: %d, VBO: %d)", p.QuadVAO, p.QuadVBO)
	gl.DeleteBuffers(1, &p.QuadVBO)
	gl.DeleteVertexArrays(1, &p.QuadVAO)
	return nil
}

// RenderTexture draws a given texture to the screen at a specified position and size.
func (p *Base) RenderTexture(shader *shaderManager.Shader, textureID uint32, position, size mgl32.Vec2, color sdl.Color) error {
	shaderProgram := shader.ProgramID
	gl.UseProgram(shaderProgram)

	// Set Uniforms
	gl.Uniform1i(gl.GetUniformLocation(shaderProgram, gl.Str("u_texture\x00")), 0)
	colorUniform := gl.GetUniformLocation(shaderProgram, gl.Str("u_textColor\x00"))
	if colorUniform == -1 {
		return fmt.Errorf("could not find 'u_textColor' uniform in shader program %d", shaderProgram)
	}
	r := float32(color.R) / 255.0
	g := float32(color.G) / 255.0
	b := float32(color.B) / 255.0
	a := float32(color.A) / 255.0
	gl.Uniform4f(colorUniform, r, g, b, a)

	model := mgl32.Ident4()
	model = model.Mul4(mgl32.Translate3D(position.X(), position.Y(), 0))
	model = model.Mul4(mgl32.Scale3D(size.X(), size.Y(), 1))
	mvp := p.Projection.Mul4(model)
	gl.UniformMatrix4fv(gl.GetUniformLocation(shaderProgram, gl.Str("u_mvpMatrix\x00")), 1, false, &mvp[0])

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, textureID)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	gl.BindVertexArray(p.QuadVAO)

	// --- THIS IS THE FIX ---
	// Force the rebinding of the data layout for this specific shader.
	// This guarantees the GPU state is correct for this draw call.
	gl.BindBuffer(gl.ARRAY_BUFFER, p.QuadVBO)
	stride := int32(4 * 4)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, stride, 0)   // Position
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, stride, 2*4) // TexCoord
	gl.EnableVertexAttribArray(0)
	gl.EnableVertexAttribArray(1)

	gl.DrawArrays(gl.TRIANGLES, 0, 6)

	// Clean up state
	gl.BindVertexArray(0)
	gl.UseProgram(0)
	return nil
}

// RenderSolidColorQuad draws a rectangle of a solid color to the screen.
func (p *Base) RenderSolidColorQuad(shader *shaderManager.Shader, position, size mgl32.Vec2, color sdl.Color) error {
	shaderProgram := shader.ProgramID
	gl.UseProgram(shaderProgram)
	// Check 1: Right after selecting the shader. This is the most common point of failure.
	// If shaderProgram is 0, this will immediately throw an INVALID_VALUE or INVALID_OPERATION error.
	if err := graphics.CheckGLError(); err != nil {
		return fmt.Errorf("RenderSolidColorQuad: error after UseProgram: %v", err)
	}

	model := mgl32.Ident4()
	model = model.Mul4(mgl32.Translate3D(position.X(), position.Y(), 0))
	model = model.Mul4(mgl32.Scale3D(size.X(), size.Y(), 1))
	mvp := p.Projection.Mul4(model)

	// Proactive Check: Ensure the uniform actually exists in the shader.
	// gl.GetUniformLocation returns -1 if not found, which doesn't set a GL error itself,
	// but will cause the next gl.Uniform call to fail.
	mvpUniform := gl.GetUniformLocation(shaderProgram, gl.Str("u_mvpMatrix\x00"))
	if mvpUniform == -1 {
		return fmt.Errorf("RenderSolidColorQuad: could not find 'u_mvpMatrix' uniform")
	}
	gl.UniformMatrix4fv(mvpUniform, 1, false, &mvp[0])

	r := float32(color.R) / 255.0
	g := float32(color.G) / 255.0
	b := float32(color.B) / 255.0
	a := float32(color.A) / 255.0
	colorUniform := gl.GetUniformLocation(shaderProgram, gl.Str("u_color\x00"))
	if colorUniform == -1 {
		return fmt.Errorf("RenderSolidColorQuad: could not find 'u_color' uniform")
	}
	gl.Uniform4f(colorUniform, r, g, b, a)

	// Check 2: After setting all uniforms. This confirms the data was sent correctly.
	if err := graphics.CheckGLError(); err != nil {
		return fmt.Errorf("RenderSolidColorQuad: error after setting uniforms: %v", err)
	}

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	gl.BindVertexArray(p.QuadVAO)
	gl.DrawArrays(gl.TRIANGLES, 0, 6)

	// Check 3: Immediately after the draw call. This is the most critical check.
	// If any state was wrong (shader, uniforms, VAO, attributes), this is where the error flag is often set.
	if err := graphics.CheckGLError(); err != nil {
		return fmt.Errorf("RenderSolidColorQuad: error after DrawArrays: %v", err)
	}

	gl.BindVertexArray(0)
	gl.Disable(gl.BLEND)
	gl.UseProgram(0)
	return nil
}

// Provide default implementations for the Page interface.
func (p *Base) HandleEvent(event sdl.Event) error { return nil }
func (p *Base) Update(dt float32) error           { return nil }
func (p *Base) Render() error                     { return nil }
