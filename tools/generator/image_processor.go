package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/lukegb/dds"
)

type TilingRule struct {
	IconWidth  int
	IconHeight int
	StepX      float64
	StepY      float64
	OffsetX    int
	OffsetY    int
}

type ImageProcessor struct {
	outDir string
	rules  map[string]TilingRule
	cache  map[string]image.Image
}

func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{
		rules: make(map[string]TilingRule),
		cache: make(map[string]image.Image),
	}
}

func (p *ImageProcessor) getRule(filename string) TilingRule {
	lowerName := strings.ToLower(filename)

	// Default: 32x32
	rule := TilingRule{IconWidth: 32, IconHeight: 32, StepX: 32, StepY: 32}

	if strings.Contains(lowerName, "gui_condition") {
		// 16x8 icons in 256x128
		// Step: 256/16 = 16, 128/8 = 16
		rule = TilingRule{IconWidth: 16, IconHeight: 16, StepX: 16, StepY: 16}
	} else if strings.Contains(lowerName, "gui_icon_skill") {
		// Most are 340x680 with 8x16 grid
		// User: 5px right shoulder, 9px bottom gap
		// StepX = (340 - 5) / 8 = 41.875
		// StepY = (680 - 9) / 16 = 41.9375
		rule = TilingRule{IconWidth: 40, IconHeight: 40, StepX: 41.875, StepY: 41.9375}

		if strings.Contains(lowerName, "skill_kmn") {
			// kmn is 340x170 (8x4 grid based on file size)
			// User: 5px shoulder, no bottom gap
			// StepY = 170 / 4 = 42.5
			rule = TilingRule{IconWidth: 40, IconHeight: 40, StepX: 41.875, StepY: 42.5}
		}
	} else if strings.Contains(lowerName, "4040icon") {
		// 3x3 grid in 128x128 (calculated)
		// Step: 128/3 = 42.66... Let's try 42.
		rule = TilingRule{IconWidth: 40, IconHeight: 40, StepX: 42, StepY: 42}
	}

	return rule
}

func (p *ImageProcessor) ExtractIcon(ddsPath, outPath string, posX, posY int) error {
	// outPath is now the full path including directory
	rule := p.getRule(filepath.Base(ddsPath))

	img, err := p.loadDDS(ddsPath)
	if err != nil {
		return err
	}

	// Calculate coordinates
	x := int(float64(rule.OffsetX) + float64(posX)*rule.StepX)
	y := int(float64(rule.OffsetY) + float64(posY)*rule.StepY)

	// Ensure we don't go out of bounds
	bounds := img.Bounds()
	if x+rule.IconWidth > bounds.Max.X {
		x = bounds.Max.X - rule.IconWidth
	}
	if y+rule.IconHeight > bounds.Max.Y {
		y = bounds.Max.Y - rule.IconHeight
	}
	if x < 0 { x = 0 }
	if y < 0 { y = 0 }

	rect := image.Rect(0, 0, rule.IconWidth, rule.IconHeight)
	dst := image.NewRGBA(rect)
	
	// Pre-check for scaling (Mabinogi uses various unscaled DDS formats like A1R5G5B5)
	// We'll check the first pixel to see if it needs normalization.
	r0, g0, b0, a0 := img.At(x, y).RGBA()
	needsScaling := (a0 > 0 && a0 <= 257) || (r0 > 0 && r0 <= 31) || (g0 > 0 && g0 <= 63) || (b0 > 0 && b0 <= 31)
	
	// Special cases that we know need scaling
	lowerPath := strings.ToLower(ddsPath)
	if strings.Contains(lowerPath, "gui_condition") || 
	   strings.Contains(lowerPath, "skill_005") || 
	   strings.Contains(lowerPath, "skill_kmn") ||
	   strings.Contains(lowerPath, "4040icon") {
		needsScaling = true
	}

	for dy := 0; dy < rule.IconHeight; dy++ {
		for dx := 0; dx < rule.IconWidth; dx++ {
			c := img.At(x+dx, y+dy)
			r, g, b, a := c.RGBA()
			
			if needsScaling {
				// Normalize Alpha: 1-bit or 8-bit unscaled
				if a == 257 || a == 1 {
					a = 65535
				} else if a > 0 && a <= 255 {
					a = a * 257
				}
				
				// Normalize Colors: raw 5-bit (31) or 6-bit (63) values
				if r <= 31 { r = r * (65535 / 31) } else if r < 256 { r = r * 257 }
				if g <= 31 { g = g * (65535 / 31) } else if g <= 63 { g = g * (65535 / 63) } else if g < 256 { g = g * 257 }
				if b <= 31 { b = b * (65535 / 31) } else if b < 256 { b = b * 257 }
			}
			
			dst.Set(dx, dy, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			})
		}
	}


	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return png.Encode(outFile, dst)
}

func (p *ImageProcessor) loadDDS(path string) (image.Image, error) {
	if img, ok := p.cache[path]; ok {
		return img, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, err := dds.Decode(file)
	if err != nil {
		return nil, err
	}

	p.cache[path] = img
	return img, nil
}
