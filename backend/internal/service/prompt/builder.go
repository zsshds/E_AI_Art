package prompt

import (
	"fmt"
	"strings"

	"github.com/imagegen/backend/internal/model"
)

type PromptBuilder struct {
	Profile *model.StyleProfile
	// Runtime overrides (from task)
	SourceImageURLs []string
	SizeOverride    string
	QualityOverride string
	ImageCount      int
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
	parts = append(parts, b.Profile.ExtraTokens...)

	stylePart := strings.Join(parts, ", ")
	cleaned := strings.TrimSpace(userInput)

	if stylePart == "" {
		return cleaned
	}
	return fmt.Sprintf("%s. %s", stylePart, cleaned)
}

// BuildChatRequest constructs a chat completions request with multimodal messages.
// This is used when source/reference images are present so the model can "read" the image content.
func (b *PromptBuilder) BuildChatRequest(userInput string) map[string]interface{} {
	profile := b.Profile
	prompt := b.Build(userInput)
	modelID := profile.Model
	if modelID == "" {
		modelID = "gpt-4o-image"
	}

	content := make([]map[string]interface{}, 0, 1+len(b.SourceImageURLs)+len(profile.ReferenceImageURLs))

	content = append(content, map[string]interface{}{
		"type": "text",
		"text": prompt,
	})

	for _, img := range b.SourceImageURLs {
		content = append(content, map[string]interface{}{
			"type":      "image_url",
			"image_url": map[string]string{"url": img},
		})
	}

	for _, refImg := range profile.ReferenceImageURLs {
		content = append(content, map[string]interface{}{
			"type":      "image_url",
			"image_url": map[string]string{"url": refImg},
		})
	}

	body := map[string]interface{}{
		"model": modelID,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": content,
			},
		},
	}

	if b.ImageCount > 1 {
		body["n"] = b.ImageCount
	}

	size := b.SizeOverride
	if size == "" {
		size = "auto"
	}
	if size != "auto" {
		body["size"] = size
	}

	return body
}

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

	size := b.SizeOverride
	if size == "" {
		size = "auto"
	}
	apiQuality := b.QualityOverride
	if apiQuality == "" {
		apiQuality = "medium"
	}

	switch model.GetModelType(modelID) {
	case model.ModelTypeGPT:
		if size != "auto" {
			body["size"] = size
		}
		if len(b.SourceImageURLs) == 1 {
			body["image"] = b.SourceImageURLs[0]
		} else if len(b.SourceImageURLs) > 1 {
			body["images"] = b.SourceImageURLs
		}
		if b.ImageCount > 1 {
			body["n"] = b.ImageCount
		}
		if len(profile.ReferenceImageURLs) > 0 {
			body["reference_images"] = profile.ReferenceImageURLs
		}
	case model.ModelTypeGemini, model.ModelTypeBanana:
		b.buildGeminiParams(body, profile, size, apiQuality)
	case model.ModelTypeMJ:
		// MJ only needs model + prompt
	}

	return body
}

func (b *PromptBuilder) buildGeminiParams(body map[string]interface{}, profile *model.StyleProfile, size, apiQuality string) {
	// Convert pixel size to aspect ratio
	aspectRatio := "1:1"
	if ar, ok := model.PixelSizeToAspectRatio[size]; ok {
		aspectRatio = ar
	}
	body["aspectRatio"] = aspectRatio

	rf := "url"
	body["response_format"] = rf

	if len(b.SourceImageURLs) == 1 {
		body["image"] = b.SourceImageURLs[0]
	} else if len(b.SourceImageURLs) > 1 {
		body["images"] = b.SourceImageURLs
	}
	if b.ImageCount > 1 {
		body["n"] = b.ImageCount
	}
	if len(profile.ReferenceImageURLs) > 0 {
		body["reference_images"] = profile.ReferenceImageURLs
	}

	// Image size only for gemini-3-pro and nano-banana-2 variants
	if strings.Contains(profile.Model, "gemini-3-pro") ||
		strings.Contains(profile.Model, "nano-banana-2") {
		if apiQuality == "high" {
			body["imageSize"] = "4K"
		} else if apiQuality == "medium" {
			body["imageSize"] = "2K"
		} else {
			body["imageSize"] = "1K"
		}
	}
}