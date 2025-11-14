package main

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// ImageGenerator handles AI image generation for game maps
type ImageGenerator struct {
	client *genai.Client
}

// NewImageGenerator creates a new image generator with the Gemini client
func NewImageGenerator(client *genai.Client) *ImageGenerator {
	return &ImageGenerator{
		client: client,
	}
}

// GenerateWorldMap generates a top-down world map image for the game
func (ig *ImageGenerator) GenerateWorldMap(ctx context.Context, worldDescription string, areas []Area) ([]byte, error) {
	// Build a description of the world layout
	areaDescriptions := ""
	for _, area := range areas {
		areaDescriptions += fmt.Sprintf("- %s: %s\n", area.ID, area.Description)
	}

	prompt := fmt.Sprintf(`Create a top-down fantasy game world map in the style of classic RPG games like Final Fantasy or Dragon Quest.

World Description:
%s

Areas in this world:
%s

Style requirements:
- Top-down bird's eye view perspective
- Fantasy/medieval aesthetic
- Clear distinct regions for each area
- Video game map style (like SNES/PS1 era RPGs)
- Use vibrant but slightly muted colors
- Show terrain features (forests, mountains, towns, etc.)
- Make it look hand-drawn/illustrated
- Include visual landmarks for each major area
- No text labels or UI elements
- Square aspect ratio suitable for game UI display

The map should show the general layout and connections between areas, giving players a sense of the world's geography.`,
		worldDescription,
		areaDescriptions,
	)

	return ig.generateImage(ctx, prompt, "world-map")
}

// GenerateZoneMap generates a detailed map for a specific zone/area
func (ig *ImageGenerator) GenerateZoneMap(ctx context.Context, zoneName string, area Area) ([]byte, error) {
	// Build description of connections
	connections := ""
	for dir, roomID := range area.Connections {
		connections += fmt.Sprintf("- %s leads to: %s\n", dir, roomID)
	}

	prompt := fmt.Sprintf(`Create a detailed top-down map of a specific zone/area in a fantasy RPG game.

Zone Name: %s
Description: %s

Connections to other areas:
%s

Style requirements:
- Top-down bird's eye view perspective
- More detailed than a world map - show specific features
- Fantasy/medieval aesthetic matching the zone's theme
- Video game zone map style (like Zelda or classic JRPGs)
- Show points of interest, obstacles, and pathways
- Use appropriate colors for the zone type (forest greens, dungeon grays, etc.)
- Hand-drawn/illustrated style
- No text labels or UI elements
- Square aspect ratio
- Show exits/connections to other areas as paths or doorways

This zone map should give players detailed navigation information for this specific area.`,
		zoneName,
		area.Description,
		connections,
	)

	return ig.generateImage(ctx, prompt, fmt.Sprintf("zone-%s", zoneName))
}

// generateImage is the core image generation function
func (ig *ImageGenerator) generateImage(ctx context.Context, prompt string, imageType string) ([]byte, error) {
	fmt.Printf("[Image Gen] Generating %s with prompt length: %d chars\n", imageType, len(prompt))

	// Use Imagen 3 model for image generation
	modelName := "imagen-3.0-generate-001"

	// Generate the image
	result, err := ig.client.Models.GenerateImages(
		ctx,
		modelName,
		prompt,
		&genai.GenerateImagesConfig{
			NumberOfImages: 1,
			AspectRatio:    "1:1", // Square for game UI
			// SafetySettings can be added here if needed
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	if len(result.GeneratedImages) == 0 {
		return nil, fmt.Errorf("no images generated")
	}

	// Get the first generated image
	generatedImage := result.GeneratedImages[0]

	// Get image data from the response
	if generatedImage.Image == nil {
		return nil, fmt.Errorf("generated image has no image data")
	}

	imageData := generatedImage.Image.ImageBytes

	if len(imageData) == 0 {
		return nil, fmt.Errorf("generated image has empty image bytes")
	}

	fmt.Printf("[Image Gen] Successfully generated %s (%d bytes)\n", imageType, len(imageData))
	return imageData, nil
}

// GenerateWorldMapWithRetry generates a world map with retry logic for failures
func (ig *ImageGenerator) GenerateWorldMapWithRetry(ctx context.Context, worldDescription string, areas []Area, maxRetries int) ([]byte, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		imageData, err := ig.GenerateWorldMap(ctx, worldDescription, areas)
		if err == nil {
			return imageData, nil
		}
		lastErr = err
		fmt.Printf("[Image Gen] Attempt %d failed: %v\n", i+1, err)
	}
	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}
