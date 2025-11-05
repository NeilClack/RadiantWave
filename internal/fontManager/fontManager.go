package fontManager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
	"radiantwavetech.com/radiant_wave/internal/config"
	"radiantwavetech.com/radiant_wave/internal/graphics"
	"radiantwavetech.com/radiant_wave/internal/logger"
	"radiantwavetech.com/radiant_wave/internal/shaderManager"
)

// FontManager handles the loading and management of fonts.
type FontManager struct {
	Fonts map[string]*FontEntry
}

var (
	fontManagerInstance *FontManager
	fontManagerOnce     sync.Once
	initErr             error // Package-level variable to capture error from within sync.Once
)

// InitFontManager initializes the FontManager singleton.
// It must be called once at startup after ShaderManager has been initialized.
func InitFontManager() error {
	fontManagerOnce.Do(func() {
		logger.LogInfo("Initializing FontManager singleton...")

		// Dependency check: Ensure ShaderManager is ready
		sm := shaderManager.Get()
		if sm == nil {
			initErr = fmt.Errorf("cannot initialize FontManager because ShaderManager is not initialized")
			return // Exit the anonymous function
		}

		// Dependency check: Ensure Config is ready
		cfg := config.Get()
		if cfg == nil {
			initErr = fmt.Errorf("cannot initialize FontManager because Config is not initialized")
			return
		}

		fm := &FontManager{
			Fonts: make(map[string]*FontEntry),
		}

		fontPath := filepath.Join(cfg.AssetsDir, "fonts")
		if err := fm.loadFonts(fontPath); err != nil {
			initErr = fmt.Errorf("could not load fonts from '%s': %w", fontPath, err)
			return
		}

		// --- NEW: Create the scrambled font after loading normal fonts ---
		// We will use "DejaVuSans" as the base. If you use a different default font,
		// change this string.
		baseFontForScramble := "Roboto-Regular"
		if _, ok := fm.Fonts[baseFontForScramble]; ok {
			if err := fm.CreateScrambledFont(baseFontForScramble, "Scrambled"); err != nil {
				// This is not a fatal error for the whole application, so we just log it.
				logger.LogWarningF("Could not create scrambled font: %v", err)
			}
		} else {
			logger.LogWarningF("Base font '%s' for scrambling not found. Skipping scrambled font creation.", baseFontForScramble)
		}
		// --- END NEW ---

		fontManagerInstance = fm
		logger.LogInfo("FontManager initialized successfully.")
	})

	// Return the error that might have been captured inside the sync.Once block.
	return initErr
}

// Get returns the singleton instance of the FontManager.
// It will panic if InitFontManager has not been called successfully first.
func Get() *FontManager {
	if fontManagerInstance == nil {
		panic("FontManager has not been initialized or initialization failed. Call InitFontManager at application startup.")
	}
	return fontManagerInstance
}

// FontEntry combines related Font information together
type FontEntry struct {
	Name string
	Font *ttf.Font
	Map  *FontMap
}

// FontMap contains the character map for a specific font
type FontMap struct {
	FontMapTextureID uint32         // Memory address of the FontMapTexture within the GPU
	Width            int32          // The width of the font map texture
	Height           int32          // The height of the font map FontMap Texture
	Glyphs           map[rune]Glyph // A map of runes and their locations within the Fontmap
	padding          int32
}

// TempGlyph is a temporary struct used only during font map creation.
type TempGlyph struct {
	Rune    rune
	Advance int
	surface *sdl.Surface
}

// Glyph contains the rendering information for a single character
type Glyph struct {
	Rune      rune     // The character this glyph represents.
	SrcRect   sdl.Rect // The source rectangle in the texture atlas.
	Advance   int      // How far to advance the cursor after drawing this glyph.
	TextureID uint32   // The ID of the texture atlas this glyph belongs to.
}

// loadFonts scans the fonts directory, loads all .ttf/.otf files, and creates a font map for each.
func (fm *FontManager) loadFonts(fontDir string) error {
	config := config.Get()
	files, err := os.ReadDir(fontDir)
	if err != nil {
		return fmt.Errorf("could not read font directory '%s': %w", fontDir, err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()
		if strings.HasSuffix(strings.ToLower(fileName), ".ttf") || strings.HasSuffix(strings.ToLower(fileName), ".otf") {
			fontPath := filepath.Join(fontDir, fileName)
			font, err := ttf.OpenFont(fontPath, int(config.BaseFontSize)) // Using a higher resolution for better quality
			if err != nil {
				logger.LogInfoF("Warning: Could not load font '%s': %v\n", fontPath, err)
				continue
			}
			fontName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
			fm.Fonts[fontName] = &FontEntry{
				Name: fontName,
				Font: font,
				Map:  nil,
			}
		}
	}

	for _, fontEntry := range fm.Fonts {
		if err := fm.createFontMap(fontEntry, false, nil); err != nil {
			return fmt.Errorf("unable to create Font Map for font: %s : %v", fontEntry.Name, err)
		}
	}

	return nil
}

// CreateScrambledFont generates a new font entry by scrambling the glyphs of a base font.
func (fm *FontManager) CreateScrambledFont(baseFontName, newFontName string) error {
	baseFont, ok := fm.Fonts[baseFontName]
	if !ok {
		return fmt.Errorf("base font '%s' not found", baseFontName)
	}

	scrambledFontEntry := &FontEntry{
		Name: newFontName,
		Font: baseFont.Font, // It uses the same underlying TTF font object for rendering
		Map:  nil,
	}

	if err := fm.createFontMap(scrambledFontEntry, true, baseFont); err != nil {
		return fmt.Errorf("unable to create scrambled font map: %w", err)
	}

	fm.Fonts[newFontName] = scrambledFontEntry
	logger.LogInfoF("Successfully created scrambled font '%s' from base '%s'\n", newFontName, baseFontName)
	return nil
}

// scrambleSurface deterministically scrambles a glyph surface using Fibonacci-based transformations.
//
// The algorithm operates in five stages:
//
//  1. Grid Division: Divides the surface into an N×N grid where N is selected from the Fibonacci
//     sequence {5, 8, 13} based on the rune value. This creates 25, 64, or 169 blocks.
//
//  2. Initial Permutation: Generates a block mapping using Fibonacci Linear Congruential
//     Generation with parameters (multiplier, increment) selected from the Fibonacci sequence
//     {13, 21, 34, 55, 89, 144}. Each block index i maps to position:
//     (multiplier × i + increment × rune) mod totalBlocks
//
//  3. Multi-Stage Transformations: Applies 3-7 sequential geometric transformations (count
//     determined by rune value). Each stage selects one of five transformation types:
//     - Type 0: Section Reversal - reverses contiguous sections of size N/2
//     - Type 1: Spiral Layer Rotation - rotates blocks within concentric square layers clockwise
//     - Type 2: Fibonacci Step Swap - swaps blocks at Fibonacci-step intervals (5, 8, or 13)
//     - Type 3: Checkerboard Transpose - conditionally transposes blocks based on (row+col) parity
//     - Type 4: Fibonacci Walk - performs pseudo-random walk swaps using Fibonacci step sizes
//
//  4. Compound Permutation: Each transformation modifies the permutation array in place,
//     creating a compound mapping that combines all applied transformations.
//
//  5. Pixel Rearrangement: Copies pixel data from source blocks to destination blocks according
//     to the final permutation, with bounds checking for partial blocks at image edges.
//
// The algorithm is deterministic: identical runes always produce identical scrambling. The
// multi-stage approach with Fibonacci-based parameters creates high-entropy scrambling that
// renders characters visually illegible while maintaining mathematical reversibility.
//
// Parameters:
//   - surface: SDL surface containing the glyph pixels (must be at least 8×8 pixels)
//   - r: rune value used as seed for all Fibonacci sequence selections and transformations
//
// Returns an error if the surface cannot be locked, nil otherwise.
func (fm *FontManager) scrambleSurface(surface *sdl.Surface, r rune) error {
	if surface.W < 8 || surface.H < 8 {
		return nil // Skip tiny glyphs
	}
	if err := surface.Lock(); err != nil {
		return fmt.Errorf("could not lock surface for scrambling: %w", err)
	}
	defer surface.Unlock()

	W := int(surface.W)
	H := int(surface.H)

	// Use much finer grid for complete illegibility
	// Fibonacci number determines grid size: 5, 8, or 13
	fib := []int{5, 8, 13}
	fibIndex := int(r) % len(fib)
	gridSize := fib[fibIndex]

	// Ensure we have at least 8x8 grid for strong scrambling
	if gridSize < 8 {
		gridSize = 8
	}

	blockW := W / gridSize
	blockH := H / gridSize

	if blockW == 0 || blockH == 0 {
		// Surface too small, fall back to 4x4
		gridSize = 4
		blockW = W / gridSize
		blockH = H / gridSize
		if blockW == 0 || blockH == 0 {
			return nil
		}
	}

	numBlocks := gridSize * gridSize

	// Create complete permutation using Fibonacci linear congruential generator
	blockIndices := make([]int, numBlocks)

	// Use Fibonacci numbers for the LCG parameters
	// F(7)=13, F(8)=21, F(9)=34, F(10)=55, F(11)=89
	fibSeq := []int{13, 21, 34, 55, 89, 144}
	multiplier := fibSeq[int(r)%len(fibSeq)]
	increment := fibSeq[(int(r)+1)%len(fibSeq)]

	// Generate pseudo-random permutation
	for i := 0; i < numBlocks; i++ {
		blockIndices[i] = (multiplier*i + increment*int(r)) % numBlocks
	}

	// Additional scrambling: apply multiple Fibonacci-based transformations
	numTransforms := 3 + (int(r) % 5) // 3-7 transformations

	for t := 0; t < numTransforms; t++ {
		transformType := (int(r) + t*fibSeq[t%len(fibSeq)]) % 5

		switch transformType {
		case 0: // Reverse sections
			sectionSize := gridSize / 2
			for i := 0; i < numBlocks; i += sectionSize {
				end := i + sectionSize
				if end > numBlocks {
					end = numBlocks
				}
				reverseSlice(blockIndices[i:end])
			}

		case 1: // Spiral rotation
			// Rotate blocks in a spiral pattern
			for layer := 0; layer < gridSize/2; layer++ {
				rotateLayer(blockIndices, gridSize, layer)
			}

		case 2: // Fibonacci walk swap
			step := fib[t%len(fib)]
			for i := 0; i < numBlocks-step; i += step {
				blockIndices[i], blockIndices[i+step] = blockIndices[i+step], blockIndices[i]
			}

		case 3: // Checkerboard transpose
			for i := 0; i < gridSize; i++ {
				for j := 0; j < gridSize; j++ {
					if (i+j)%2 == 0 {
						idx1 := i*gridSize + j
						idx2 := j*gridSize + i
						if idx2 < numBlocks {
							blockIndices[idx1], blockIndices[idx2] = blockIndices[idx2], blockIndices[idx1]
						}
					}
				}
			}

		case 4: // Random-walk permutation
			walkLen := fib[(t+1)%len(fib)]
			pos := int(r+rune(t)) % numBlocks
			for w := 0; w < walkLen && pos < numBlocks-1; w++ {
				nextPos := (pos + fib[w%len(fib)]) % numBlocks
				blockIndices[pos], blockIndices[nextPos] = blockIndices[nextPos], blockIndices[pos]
				pos = nextPos
			}
		}
	}

	// Copy pixels to perform the scrambling
	pixels := surface.Pixels()
	bytesPerPixel := int(surface.Format.BytesPerPixel)
	pitch := int(surface.Pitch)
	originalPixels := make([]byte, len(pixels))
	copy(originalPixels, pixels)

	// Rearrange all blocks
	for dstIdx := 0; dstIdx < numBlocks; dstIdx++ {
		srcIdx := blockIndices[dstIdx]

		srcCol := srcIdx % gridSize
		srcRow := srcIdx / gridSize
		dstCol := dstIdx % gridSize
		dstRow := dstIdx / gridSize

		srcX := srcCol * blockW
		srcY := srcRow * blockH
		dstX := dstCol * blockW
		dstY := dstRow * blockH

		// Copy block pixel by pixel
		for y := 0; y < blockH; y++ {
			if srcY+y >= H || dstY+y >= H {
				continue
			}

			srcOffset := (srcY+y)*pitch + srcX*bytesPerPixel
			dstOffset := (dstY+y)*pitch + dstX*bytesPerPixel

			rowLen := blockW * bytesPerPixel
			if srcX+blockW > W {
				rowLen = (W - srcX) * bytesPerPixel
			}
			if dstX+blockW > W {
				maxLen := (W - dstX) * bytesPerPixel
				if maxLen < rowLen {
					rowLen = maxLen
				}
			}

			if srcOffset+rowLen <= len(originalPixels) && dstOffset+rowLen <= len(pixels) {
				copy(pixels[dstOffset:dstOffset+rowLen], originalPixels[srcOffset:srcOffset+rowLen])
			}
		}
	}

	return nil
}

// Helper function to reverse a slice
func reverseSlice(arr []int) {
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
}

// Helper function to rotate a layer in spiral pattern
func rotateLayer(arr []int, gridSize, layer int) {
	if layer >= gridSize/2 {
		return
	}

	// Extract the layer elements
	var temp []int

	// Top edge
	for i := layer; i < gridSize-layer; i++ {
		temp = append(temp, arr[layer*gridSize+i])
	}
	// Right edge
	for i := layer + 1; i < gridSize-layer; i++ {
		temp = append(temp, arr[i*gridSize+(gridSize-layer-1)])
	}
	// Bottom edge
	if gridSize-layer-1 > layer {
		for i := gridSize - layer - 2; i >= layer; i-- {
			temp = append(temp, arr[(gridSize-layer-1)*gridSize+i])
		}
	}
	// Left edge
	if gridSize-layer-1 > layer {
		for i := gridSize - layer - 2; i > layer; i-- {
			temp = append(temp, arr[i*gridSize+layer])
		}
	}

	// Rotate the temp array by 1 position
	if len(temp) > 0 {
		last := temp[len(temp)-1]
		copy(temp[1:], temp[0:len(temp)-1])
		temp[0] = last
	}

	// Put back the rotated elements
	idx := 0
	// Top edge
	for i := layer; i < gridSize-layer && idx < len(temp); i++ {
		arr[layer*gridSize+i] = temp[idx]
		idx++
	}
	// Right edge
	for i := layer + 1; i < gridSize-layer && idx < len(temp); i++ {
		arr[i*gridSize+(gridSize-layer-1)] = temp[idx]
		idx++
	}
	// Bottom edge
	if gridSize-layer-1 > layer {
		for i := gridSize - layer - 2; i >= layer && idx < len(temp); i-- {
			arr[(gridSize-layer-1)*gridSize+i] = temp[idx]
			idx++
		}
	}
	// Left edge
	if gridSize-layer-1 > layer {
		for i := gridSize - layer - 2; i > layer && idx < len(temp); i-- {
			arr[i*gridSize+layer] = temp[idx]
			idx++
		}
	}
}

// createFontMap generates a texture atlas for a given font entry.
func (fm *FontManager) createFontMap(font *FontEntry, scramble bool, baseFont *FontEntry) error {
	var chars []rune
	for i := rune(32); i <= 126; i++ { // Common ASCII characters
		chars = append(chars, i)
	}

	tempGlyphs := make([]TempGlyph, 0, len(chars))
	var totalWidth, maxHeight int32
	padding := int32(2) // 2 pixels of padding between glyphs

	// If scrambling, use the base font's metrics but our new entry's name
	fontToRenderFrom := font
	if scramble && baseFont != nil {
		fontToRenderFrom = baseFont
	}

	for _, char := range chars {
		tempGlyph, err := fm.createTempGlyph(fontToRenderFrom, char)
		if err != nil {
			logger.LogInfoF("Warning: could not create glyph for char '%c' in font %s: %v\n", char, font.Name, err)
			continue
		}

		// --- NEW: Scramble the surface if requested ---
		if scramble {
			if err := fm.scrambleSurface(tempGlyph.surface, char); err != nil {
				logger.LogWarningF("Could not scramble glyph for '%c': %v", char, err)
				// We can continue, it will just use the unscrambled glyph
			}
		}
		// --- END NEW ---

		tempGlyphs = append(tempGlyphs, tempGlyph)
		totalWidth += tempGlyph.surface.W + padding
		if tempGlyph.surface.H > maxHeight {
			maxHeight = tempGlyph.surface.H
		}
	}

	atlasSurface, err := sdl.CreateRGBSurface(0, totalWidth, maxHeight, 32, 0x00ff0000, 0x0000ff00, 0x000000ff, 0xff000000)
	if err != nil {
		return fmt.Errorf("could not create atlas surface: %w", err)
	}
	defer atlasSurface.Free()

	fontMap := &FontMap{
		Glyphs:  make(map[rune]Glyph),
		padding: padding,
	}

	var currentX int32 = 0
	for _, tempGlyph := range tempGlyphs {
		dstRect := sdl.Rect{X: currentX, Y: 0, W: tempGlyph.surface.W, H: tempGlyph.surface.H}
		if err := tempGlyph.surface.Blit(nil, atlasSurface, &dstRect); err != nil {
			tempGlyph.surface.Free()
			logger.LogInfoF("Warning: Failed to blit glyph '%c': %v\n", tempGlyph.Rune, err)
			continue
		}

		fontMap.Glyphs[tempGlyph.Rune] = Glyph{
			Rune:    tempGlyph.Rune,
			Advance: tempGlyph.Advance,
			SrcRect: dstRect,
		}

		currentX += tempGlyph.surface.W + padding
		tempGlyph.surface.Free()
	}

	textureID, err := graphics.CreateTextureFromSurface(atlasSurface)
	if err != nil {
		return fmt.Errorf("could not create texture from atlas surface: %w", err)
	}

	fontMap.FontMapTextureID = textureID
	fontMap.Width = atlasSurface.W
	fontMap.Height = atlasSurface.H

	for r, g := range fontMap.Glyphs {
		g.TextureID = textureID
		fontMap.Glyphs[r] = g
	}

	font.Map = fontMap
	return nil
}

// createTempGlyph renders a single character to a temporary SDL surface.
func (fm *FontManager) createTempGlyph(font *FontEntry, char rune) (TempGlyph, error) {
	runeSurface, err := font.Font.RenderGlyphBlended(char, sdl.Color{R: 255, G: 255, B: 255, A: 255})
	if err != nil {
		return TempGlyph{}, fmt.Errorf("unable to create glyph for char '%c': %w", char, err)
	}

	metrics, err := font.Font.GlyphMetrics(char)
	if err != nil {
		runeSurface.Free()
		return TempGlyph{}, fmt.Errorf("unable to fetch glyph metrics for char '%c': %w", char, err)
	}

	return TempGlyph{
		Rune:    char,
		Advance: metrics.Advance,
		surface: runeSurface,
	}, nil
}

// GetFont retrieves a loaded font by its name.
func (fm *FontManager) GetFont(name string) (*FontEntry, bool) {
	fontEntry, ok := fm.Fonts[name]
	return fontEntry, ok
}

// GetGlyph retrieves the rendering data for a specific character from a font's map.
func (fm *FontManager) GetGlyph(fontName string, r rune) (Glyph, bool) {
	fontEntry, ok := fm.Fonts[fontName]
	if !ok || fontEntry.Map == nil {
		return Glyph{}, false
	}
	glyph, ok := fontEntry.Map.Glyphs[r]
	return glyph, ok
}

// GetFontNames returns a slice of all loaded font names.
func (fm *FontManager) GetFontNames() []string {
	names := make([]string, 0, len(fm.Fonts))
	for name := range fm.Fonts {
		names = append(names, name)
	}
	return names
}

// Close frees all loaded font resources, including the generated textures.
func (fm *FontManager) Close() {
	for _, fontEntry := range fm.Fonts {
		if fontEntry.Map != nil && fontEntry.Map.FontMapTextureID != 0 {
			gl.DeleteTextures(1, &fontEntry.Map.FontMapTextureID)
		}
		if fontEntry.Font != nil {
			fontEntry.Font.Close()
		}
	}
}

func (fm *FontManager) CreateStringTexture(fontName string, text string, shader *shaderManager.Shader) (textureID uint32, width int32, height int32, err error) {
	shaderID := shader.ProgramID
	if fontName == "" || text == "" {
		return 0, 0, 0, fmt.Errorf("font name and text cannot be blank")
	}

	selectedFont, ok := fm.Fonts[fontName]
	if !ok || selectedFont.Map == nil {
		return 0, 0, 0, fmt.Errorf("font '%s' not found or its font map is not created", fontName)
	}

	var totalWidth, maxHeight int32
	glyphsToRender := make([]Glyph, 0, len(text))

	for _, r := range text {
		g, ok := selectedFont.Map.Glyphs[r]
		if !ok {
			g = selectedFont.Map.Glyphs['?'] // Fallback
		}
		totalWidth += g.SrcRect.W + selectedFont.Map.padding
		if g.SrcRect.H > maxHeight {
			maxHeight = g.SrcRect.H
		}
		glyphsToRender = append(glyphsToRender, g)
	}

	if totalWidth <= 0 || maxHeight <= 0 {
		return 0, 0, 0, fmt.Errorf("calculated texture dimensions are invalid: %dx%d", totalWidth, maxHeight)
	}

	var lastFBO int32
	gl.GetIntegerv(gl.FRAMEBUFFER_BINDING, &lastFBO)
	var lastViewport [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &lastViewport[0])

	var fboID uint32
	gl.GenFramebuffers(1, &fboID)
	gl.BindFramebuffer(gl.FRAMEBUFFER, fboID)

	var newTextureID uint32
	gl.GenTextures(1, &newTextureID)
	gl.BindTexture(gl.TEXTURE_2D, newTextureID)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	// Optional alignment fix:
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)

	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, totalWidth, maxHeight, 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, newTextureID, 0)

	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(lastFBO))
		gl.DeleteFramebuffers(1, &fboID)
		gl.DeleteTextures(1, &newTextureID)
		return 0, 0, 0, fmt.Errorf("framebuffer is not complete")
	}

	gl.Viewport(0, 0, totalWidth, maxHeight)
	gl.ClearColor(0.0, 0.0, 0.0, 0.0)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.UseProgram(shaderID)

	projection := mgl32.Ortho(0, float32(totalWidth), 0, float32(maxHeight), -1, 1)
	mvpUniform := gl.GetUniformLocation(shaderID, gl.Str("u_mvpMatrix\x00"))
	gl.UniformMatrix4fv(mvpUniform, 1, false, &projection[0])

	// Set text color to white so glyphs render correctly into the texture.
	textColorUniform := gl.GetUniformLocation(shaderID, gl.Str("u_textColor\x00"))
	if textColorUniform != -1 {
		gl.Uniform4f(textColorUniform, 1.0, 1.0, 1.0, 1.0)
	}

	gl.ActiveTexture(gl.TEXTURE0)
	textureUniform := gl.GetUniformLocation(shaderID, gl.Str("u_texture\x00"))
	gl.Uniform1i(textureUniform, 0)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.BindTexture(gl.TEXTURE_2D, selectedFont.Map.FontMapTextureID)

	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)
	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 6*4*4, nil, gl.DYNAMIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, 4*4, 0)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 4*4, 2*4)

	var currentX float32
	atlasW, atlasH := float32(selectedFont.Map.Width), float32(selectedFont.Map.Height)

	for _, glyph := range glyphsToRender {
		xPos := currentX
		yPos := float32(0)
		w, h := float32(glyph.SrcRect.W), float32(glyph.SrcRect.H)
		u0 := float32(glyph.SrcRect.X) / atlasW
		v0 := float32(glyph.SrcRect.Y) / atlasH
		u1 := u0 + w/atlasW
		v1 := v0 + h/atlasH
		vertices := []float32{
			xPos, yPos + h, u0, v0,
			xPos, yPos, u0, v1,
			xPos + w, yPos, u1, v1,
			xPos, yPos + h, u0, v0,
			xPos + w, yPos, u1, v1,
			xPos + w, yPos + h, u1, v0,
		}
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(vertices)*4, gl.Ptr(vertices))
		gl.DrawArrays(gl.TRIANGLES, 0, 6)
		currentX += float32(glyph.Advance)
	}

	gl.BindVertexArray(0)
	gl.DeleteVertexArrays(1, &vao)
	gl.DeleteBuffers(1, &vbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(lastFBO))
	gl.Viewport(lastViewport[0], lastViewport[1], lastViewport[2], lastViewport[3])
	gl.DeleteFramebuffers(1, &fboID)
	gl.Disable(gl.BLEND)

	return newTextureID, totalWidth, maxHeight, nil
}

// NewCreateStringTexture does the same thing as CreateStringTexture, except that it returns the width in pixels used by the glyphs
// in the texture, rather than the width of the texture itself. This can be used for trimming.
func (fm *FontManager) NewCreateStringTexture(fontName string, text string, shader *shaderManager.Shader) (textureID uint32, width float32, height int32, err error) {
	shaderID := shader.ProgramID
	if fontName == "" || text == "" {
		return 0, 0, 0, fmt.Errorf("font name and text cannot be blank")
	}

	selectedFont, ok := fm.Fonts[fontName]
	if !ok || selectedFont.Map == nil {
		return 0, 0, 0, fmt.Errorf("font '%s' not found or its font map is not created", fontName)
	}

	var totalWidth, maxHeight int32
	glyphsToRender := make([]Glyph, 0, len(text))

	for _, r := range text {
		g, ok := selectedFont.Map.Glyphs[r]
		if !ok {
			g = selectedFont.Map.Glyphs['?'] // Fallback
		}
		totalWidth += g.SrcRect.W + selectedFont.Map.padding
		if g.SrcRect.H > maxHeight {
			maxHeight = g.SrcRect.H
		}
		glyphsToRender = append(glyphsToRender, g)
	}

	if totalWidth <= 0 || maxHeight <= 0 {
		return 0, 0, 0, fmt.Errorf("calculated texture dimensions are invalid: %dx%d", totalWidth, maxHeight)
	}

	var lastFBO int32
	gl.GetIntegerv(gl.FRAMEBUFFER_BINDING, &lastFBO)
	var lastViewport [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &lastViewport[0])

	var fboID uint32
	gl.GenFramebuffers(1, &fboID)
	gl.BindFramebuffer(gl.FRAMEBUFFER, fboID)

	var newTextureID uint32
	gl.GenTextures(1, &newTextureID)
	gl.BindTexture(gl.TEXTURE_2D, newTextureID)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	// Optional alignment fix:
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)

	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, totalWidth, maxHeight, 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, newTextureID, 0)

	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(lastFBO))
		gl.DeleteFramebuffers(1, &fboID)
		gl.DeleteTextures(1, &newTextureID)
		return 0, 0, 0, fmt.Errorf("framebuffer is not complete")
	}

	gl.Viewport(0, 0, totalWidth, maxHeight)
	gl.ClearColor(0.0, 0.0, 0.0, 0.0)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.UseProgram(shaderID)

	projection := mgl32.Ortho(0, float32(totalWidth), 0, float32(maxHeight), -1, 1)
	mvpUniform := gl.GetUniformLocation(shaderID, gl.Str("u_mvpMatrix\x00"))
	gl.UniformMatrix4fv(mvpUniform, 1, false, &projection[0])

	// Set text color to white so glyphs render correctly into the texture.
	textColorUniform := gl.GetUniformLocation(shaderID, gl.Str("u_textColor\x00"))
	if textColorUniform != -1 {
		gl.Uniform4f(textColorUniform, 1.0, 1.0, 1.0, 1.0)
	}

	gl.ActiveTexture(gl.TEXTURE0)
	textureUniform := gl.GetUniformLocation(shaderID, gl.Str("u_texture\x00"))
	gl.Uniform1i(textureUniform, 0)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.BindTexture(gl.TEXTURE_2D, selectedFont.Map.FontMapTextureID)

	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)
	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 6*4*4, nil, gl.DYNAMIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, 4*4, 0)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 4*4, 2*4)

	var currentX float32
	atlasW, atlasH := float32(selectedFont.Map.Width), float32(selectedFont.Map.Height)

	for _, glyph := range glyphsToRender {
		xPos := currentX
		yPos := float32(0)
		w, h := float32(glyph.SrcRect.W), float32(glyph.SrcRect.H)
		u0 := float32(glyph.SrcRect.X) / atlasW
		v0 := float32(glyph.SrcRect.Y) / atlasH
		u1 := u0 + w/atlasW
		v1 := v0 + h/atlasH
		vertices := []float32{
			xPos, yPos + h, u0, v0,
			xPos, yPos, u0, v1,
			xPos + w, yPos, u1, v1,
			xPos, yPos + h, u0, v0,
			xPos + w, yPos, u1, v1,
			xPos + w, yPos + h, u1, v0,
		}
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(vertices)*4, gl.Ptr(vertices))
		gl.DrawArrays(gl.TRIANGLES, 0, 6)
		currentX += float32(glyph.Advance)
	}

	usedWidth := float32(currentX)

	gl.BindVertexArray(0)
	gl.DeleteVertexArrays(1, &vao)
	gl.DeleteBuffers(1, &vbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(lastFBO))
	gl.Viewport(lastViewport[0], lastViewport[1], lastViewport[2], lastViewport[3])
	gl.DeleteFramebuffers(1, &fboID)
	gl.Disable(gl.BLEND)

	return newTextureID, usedWidth, maxHeight, nil
}
