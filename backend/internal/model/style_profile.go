package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StyleProfile struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name      string             `json:"name" bson:"name"`
	CreatedBy string             `json:"created_by" bson:"created_by"`
	Version   int                `json:"version" bson:"version"`
	IsLocked  bool               `json:"is_locked" bson:"is_locked"`

	// Prompt 语义层
	ArtStyle    string   `json:"art_style" bson:"art_style"`
	ColorTone   string   `json:"color_tone" bson:"color_tone"`
	Lighting    string   `json:"lighting" bson:"lighting"`
	QualityTags string   `json:"quality_tags" bson:"quality_tags"`
	Composition string   `json:"composition" bson:"composition"`
	ExtraTokens []string `json:"extra_tokens" bson:"extra_tokens"`

	// 模型选择
	Model string `json:"model" bson:"model"`

	// API 参数层
	Size         string `json:"size" bson:"size"`           // GPT: size preset key | Gemini: aspect_ratio
	APIQuality   string `json:"api_quality" bson:"api_quality"`
	Background   string `json:"background" bson:"background"`
	OutputFormat string `json:"output_format" bson:"output_format"`
	Compression  int    `json:"compression" bson:"compression"`

	// 风格参考图
	ReferenceImageURL string `json:"reference_image_url" bson:"reference_image_url"`

	// 锁定后的 Prompt 模板
	LockedPromptPrefix string    `json:"locked_prompt_prefix" bson:"locked_prompt_prefix"`
	CreatedAt          time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" bson:"updated_at"`
}

// Model type constants
type ModelType string

const (
	ModelTypeGPT    ModelType = "gpt"
	ModelTypeGemini ModelType = "gemini"
	ModelTypeBanana ModelType = "banana"
	ModelTypeMJ     ModelType = "mj"
)

// Supported models
var SupportedModels = map[string]ModelType{
	"gpt-4o-image":               ModelTypeGPT,
	"gpt-4o-image-vip":           ModelTypeGPT,
	"gpt-image-2":                ModelTypeGPT,
	"gpt-image-1.5":              ModelTypeGPT,
	"gemini-3-pro-image-preview": ModelTypeGemini,
	"gemini-2.5-flash-image":     ModelTypeGemini,
	"nano-banana-pro":            ModelTypeGemini,
	"nano-banana-2":              ModelTypeBanana,
	"nano-banana-2-2k":           ModelTypeBanana,
	"nano-banana-2-4k":           ModelTypeBanana,
	"mj_imagine":                 ModelTypeMJ,
}

// GPT model size presets (key -> "WxH")
var SizePresets = map[string]string{
	"square_1k":    "1024x1024",
	"landscape_hd": "1536x1024",
	"portrait_hd":  "1024x1536",
}

// Gemini model aspect ratios
var AspectRatioOptions = []string{
	"1:1", "4:3", "3:4", "16:9", "9:16", "2:3", "3:2", "4:5", "5:4", "21:9",
}

// Gemini image sizes (only for gemini-3-pro-image-preview, nano-banana-2)
var GeminiImageSizes = []string{"1K", "2K", "4K"}

// GetModelType returns the ModelType for a given model ID
func GetModelType(modelID string) ModelType {
	if t, ok := SupportedModels[modelID]; ok {
		return t
	}
	return ModelTypeGPT // default
}

// IsGeminiModel checks if model supports aspect_ratio + image_size params
func IsGeminiModel(modelID string) bool {
	return GetModelType(modelID) == ModelTypeGemini
}

// IsGPTModel checks if model uses standard size params
func IsGPTModel(modelID string) bool {
	return GetModelType(modelID) == ModelTypeGPT
}

// Image generation model keywords for filtering /v1/models response
var imageModelKeywords = []string{
	"image", "banana", "dall-e", "imagen", "flux",
	"mj_", "midjourney", "sdxl", "stable-diffusion",
}

// IsImageModel checks if a model ID looks like an image generation model.
func IsImageModel(modelID string) bool {
	for _, kw := range imageModelKeywords {
		if len(modelID) >= len(kw) {
			// Case-insensitive substring match
			for i := 0; i <= len(modelID)-len(kw); i++ {
				if eqFold(modelID[i:i+len(kw)], kw) {
					return true
				}
			}
		}
	}
	return false
}

// Case-insensitive ASCII comparison
func eqFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
