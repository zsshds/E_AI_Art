package prompt

import (
	"fmt"
	"strings"

	"github.com/imagegen/backend/internal/model"
)

type PromptBuilder struct {
	Profile *model.StyleProfile
}

func NewPromptBuilder(profile *model.StyleProfile) *PromptBuilder {
	return &PromptBuilder{Profile: profile}
}

func (b *PromptBuilder) Build(userInput string) string {
	parts := []string{}
	appendIfNotEmpty := func(s string) {
		if strings.TrimSpace(s) != "" {
			parts = append(parts, s)
		}
	}

	appendIfNotEmpty(b.Profile.ArtStyle)
	appendIfNotEmpty(b.Profile.ColorTone)
	appendIfNotEmpty(b.Profile.Lighting)
	appendIfNotEmpty(b.Profile.QualityTags)
	appendIfNotEmpty(b.Profile.Composition)
	parts = append(parts, b.Profile.ExtraTokens...)

	stylePart := strings.Join(parts, ", ")
	cleaned := sanitizeUserInput(userInput)

	if stylePart == "" {
		return cleaned
	}
	return fmt.Sprintf("%s. %s", stylePart, cleaned)
}

// BuildAPIRequest constructs the API request body appropriate for the selected model.
//
// GPT models (gpt-4o-image, gpt-image-1.5):
//   - model, prompt, n, size, quality, response_format
//
// Gemini models (gemini-*, nano-banana-*):
//   - model, prompt, aspect_ratio, response_format, image (optional), image_size (optional)
//
// MJ models (mj_imagine):
//   - model, prompt
func (b *PromptBuilder) BuildAPIRequest(userInput string) map[string]interface{} {
	profile := b.Profile
	prompt := b.Build(userInput)
	modelID := profile.Model
	if modelID == "" {
		modelID = "gpt-4o-image"
	}

	body := map[string]interface{}{
		"model":  modelID,
		"prompt": prompt,
	}

	switch model.GetModelType(modelID) {
	case model.ModelTypeGPT:
		b.buildGPTParams(body, profile)
	case model.ModelTypeGemini:
		b.buildGeminiParams(body, profile)
	case model.ModelTypeMJ:
		// MJ only needs model + prompt
	}

	return body
}

func (b *PromptBuilder) buildGPTParams(body map[string]interface{}, profile *model.StyleProfile) {
	size := profile.Size
	if preset, ok := model.SizePresets[size]; ok {
		size = preset
	}
	body["n"] = 1
	body["size"] = size
	body["quality"] = profile.APIQuality
	body["response_format"] = profile.OutputFormat
}

func (b *PromptBuilder) buildGeminiParams(body map[string]interface{}, profile *model.StyleProfile) {
	aspectRatio := profile.Size
	// For Gemini, Size field holds the aspect_ratio
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}
	body["aspect_ratio"] = aspectRatio
	body["response_format"] = profile.OutputFormat

	if profile.ReferenceImageURL != "" {
		body["image"] = profile.ReferenceImageURL
	}

	// Image size only for gemini-3-pro and nano-banana-2 variants
	if strings.Contains(profile.Model, "gemini-3-pro") ||
		strings.Contains(profile.Model, "nano-banana-2") {
		// Use api_quality as image_size hint (1K/2K/4K) or default to 1K
		if profile.APIQuality == "high" {
			body["image_size"] = "4K"
		} else if profile.APIQuality == "medium" {
			body["image_size"] = "2K"
		} else {
			body["image_size"] = "1K"
		}
	}
}

func sanitizeUserInput(input string) string {
	blocked := []string{
		"realistic", "photorealistic", "3d render", "3d cg",
		"oil painting", "watercolor", "sketch", "pixel art",
	}
	result := input
	for _, word := range blocked {
		result = strings.ReplaceAll(strings.ToLower(result), word, "")
	}
	return strings.TrimSpace(result)
}
