package main

import (
	"context"
	"encoding/json"
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
	// Build a description of the world layout with coordinates
	// Normalize coordinates to a 0-1000 range for the AI
	minX, minY, maxX, maxY := float64(0), float64(0), float64(0), float64(0)
	for i, area := range areas {
		if i == 0 {
			minX, minY, maxX, maxY = area.Coordinates.X, area.Coordinates.Y, area.Coordinates.X, area.Coordinates.Y
		} else {
			if area.Coordinates.X < minX {
				minX = area.Coordinates.X
			}
			if area.Coordinates.X > maxX {
				maxX = area.Coordinates.X
			}
			if area.Coordinates.Y < minY {
				minY = area.Coordinates.Y
			}
			if area.Coordinates.Y > maxY {
				maxY = area.Coordinates.Y
			}
		}
	}

	rangeX := maxX - minX
	rangeY := maxY - minY
	if rangeX == 0 {
		rangeX = 1
	}
	if rangeY == 0 {
		rangeY = 1
	}

	areaDescriptions := ""
	for _, area := range areas {
		// Normalize to 0-1000 range
		normalizedX := ((area.Coordinates.X - minX) / rangeX) * 1000
		normalizedY := ((area.Coordinates.Y - minY) / rangeY) * 1000

		// Build connection info
		connections := ""
		for dir, connID := range area.Connections {
			connections += fmt.Sprintf("%s to %s, ", dir, connID)
		}
		if connections != "" {
			connections = connections[:len(connections)-2] // Remove trailing ", "
		}

		areaDescriptions += fmt.Sprintf("- %s at position (%.0f, %.0f): %s. Connections: %s\n",
			area.ID, normalizedX, normalizedY, area.Description, connections)
	}

	prompt := fmt.Sprintf(`Create a medieval fantasy overworld map image.

World Description:
%s

Areas in this world:
%s

STYLE REQUIREMENTS:
- Medieval fantasy theme (castles, forests, mountains, villages, dungeons, etc.)
- Overworld/world map perspective - like looking down at a game world
- Use clear, distinct visual landmarks for areas of interest
- Draw paths or roads connecting different areas
- Rich, atmospheric fantasy art style
- Square 1:1 aspect ratio (1024x1024)
- NO text labels or area names (overlays will be added separately)
- Make each location visually distinctive so they can be identified

Create an immersive fantasy world map that captures the adventure and atmosphere of the setting.`,
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

	// Use Nano Banana (Gemini 2.5 Flash Image) for image generation
	modelName := "gemini-2.5-flash-preview-image-generation"

	// Generate the image using GenerateContent API with image modality
	result, err := ig.client.Models.GenerateContent(
		ctx,
		modelName,
		[]*genai.Content{
			{
				Role: "user",
				Parts: []*genai.Part{
					{Text: prompt},
				},
			},
		},
		&genai.GenerateContentConfig{
			ResponseModalities: []string{"IMAGE", "TEXT"},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no images generated")
	}

	// Find the image part in the response
	for _, part := range result.Candidates[0].Content.Parts {
		if part.InlineData != nil && part.InlineData.Data != nil {
			imageData := part.InlineData.Data
			if len(imageData) == 0 {
				continue
			}
			fmt.Printf("[Image Gen] Successfully generated %s (%d bytes)\n", imageType, len(imageData))
			return imageData, nil
		}
	}

	return nil, fmt.Errorf("no image data found in response")
}

// ExtractCoordinatesFromImage uses AI vision to analyze the map image and extract pixel coordinates for each area
func (ig *ImageGenerator) ExtractCoordinatesFromImage(ctx context.Context, imageData []byte, areas []Area) (map[string]PixelCoordinates, error) {
	fmt.Printf("[Image Gen] Analyzing map image to extract coordinates for %d areas\n", len(areas))

	// Build a description of areas to identify
	areaDescriptions := ""
	for _, area := range areas {
		connections := ""
		for dir, connID := range area.Connections {
			connections += fmt.Sprintf("%s to %s, ", dir, connID)
		}
		if connections != "" {
			connections = connections[:len(connections)-2] // Remove trailing ", "
		}
		areaDescriptions += fmt.Sprintf("- Area ID: %s\n  Description: %s\n  Connections: %s\n\n",
			area.ID, area.Description, connections)
	}

	prompt := fmt.Sprintf(`You are analyzing a fantasy world map image to identify the pixel coordinates of each location.

The image is 1024x1024 pixels. The coordinate system has (0, 0) at the TOP-LEFT corner, X increases to the RIGHT, and Y increases DOWNWARD.

Here are the areas/locations you need to find in the image:

%s

For each area, identify the most visually distinctive landmark or feature that represents it in the image, and provide its approximate CENTER pixel coordinates.

Respond ONLY with a JSON object in this exact format (no markdown, no code blocks):
{
  "area-id-1": {"x": 100, "y": 200},
  "area-id-2": {"x": 300, "y": 400}
}

Be as accurate as possible with pixel positions. Look for distinctive visual markers like castles, forests, mountains, villages, dungeons, etc.`, areaDescriptions)

	// Use Gemini vision model with GenerateContent API
	modelName := "gemini-2.0-flash-exp"

	// Create a vision request with the image and prompt
	result, err := ig.client.Models.GenerateContent(
		ctx,
		modelName,
		[]*genai.Content{
			{
				Role: "user",
				Parts: []*genai.Part{
					{InlineData: &genai.Blob{
						MIMEType: "image/png",
						Data:     imageData,
					}},
					{Text: prompt},
				},
			},
		},
		&genai.GenerateContentConfig{
			Temperature: genai.Ptr[float32](0.1), // Low temperature for consistency
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze image: %w", err)
	}

	// Extract text response
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from vision model")
	}

	responseText := ""
	for _, part := range result.Candidates[0].Content.Parts {
		if part.Text != "" {
			responseText += part.Text
		}
	}

	fmt.Printf("[Image Gen] Vision analysis response: %s\n", responseText)

	// Parse JSON response
	var coordinates map[string]PixelCoordinates
	err = json.Unmarshal([]byte(responseText), &coordinates)
	if err != nil {
		return nil, fmt.Errorf("failed to parse coordinate response: %w", err)
	}

	fmt.Printf("[Image Gen] Successfully extracted coordinates for %d areas\n", len(coordinates))
	return coordinates, nil
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
