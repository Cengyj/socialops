package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

const (
	socialTaskAvatarImageWidth  = 400
	socialTaskAvatarImageHeight = 400
	socialTaskBannerImageWidth  = 1500
	socialTaskBannerImageHeight = 500
	socialTaskPostMediaMaxItems = 4

	socialTaskMediaSourceUnsupportedMessage = "%s media source is not supported for SocialOps execution"
	socialTaskVideoUnsupportedMessage       = "video media is not supported for SocialOps execution"
)

func validateSocialTaskImageMedia(label string, ref *SocialTaskMediaRef) error {
	if ref == nil || ref.IsZero() {
		return fmt.Errorf("%s media is required", label)
	}
	contentType := socialTaskMediaContentType(ref)
	if contentType != "" && !strings.HasPrefix(contentType, "image/") {
		return fmt.Errorf("%s media must be an image", label)
	}
	rawURL := strings.TrimSpace(ref.URL)
	if strings.HasPrefix(strings.ToLower(rawURL), "data:") {
		if _, _, _, err := socialTaskMediaDimensions(ref); err != nil {
			return fmt.Errorf("%s media is invalid", label)
		}
	}
	return nil
}

func validateSocialTaskExactImageDimensions(label string, ref *SocialTaskMediaRef, requiredWidth, requiredHeight int) error {
	width, height, known, err := socialTaskMediaDimensions(ref)
	if err != nil {
		return fmt.Errorf("%s media is invalid", label)
	}
	if !known || width != requiredWidth || height != requiredHeight {
		return fmt.Errorf("%s image must be %dx%d pixels", label, requiredWidth, requiredHeight)
	}
	return nil
}

func validateSocialTaskExecutableInlineMediaSource(label string, ref *SocialTaskMediaRef) error {
	if ref == nil || ref.IsZero() {
		return nil
	}
	source := strings.TrimSpace(ref.Source)
	if socialTaskMediaRefExecutableStored(ref) {
		return nil
	}
	if source != "" && source != "inline" {
		return fmt.Errorf(socialTaskMediaSourceUnsupportedMessage, label)
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(ref.URL)), "data:") {
		return fmt.Errorf(socialTaskMediaSourceUnsupportedMessage, label)
	}
	return nil
}

func socialTaskMediaRefExecutableStored(ref *SocialTaskMediaRef) bool {
	if ref == nil || ref.IsZero() {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(ref.Source), "library") {
		return false
	}
	storageKey := strings.ToLower(strings.TrimSpace(ref.StorageKey))
	if storageKey == "" {
		return false
	}
	return strings.HasPrefix(storageKey, "social-task/")
}

func validateSocialTaskSupportedPostMedia(refs []SocialTaskMediaRef) error {
	if len(refs) > socialTaskPostMediaMaxItems {
		return fmt.Errorf("post media cannot exceed %d items", socialTaskPostMediaMaxItems)
	}
	for i, ref := range refs {
		label := fmt.Sprintf("post media #%d", i+1)
		if err := validateSocialTaskExecutableInlineMediaSource(label, &ref); err != nil {
			return err
		}
		contentType := socialTaskMediaContentType(&ref)
		switch {
		case strings.HasPrefix(contentType, "video/"):
			return fmt.Errorf(socialTaskVideoUnsupportedMessage)
		case contentType == "" || !strings.HasPrefix(contentType, "image/"):
			return fmt.Errorf("post media content type is not supported")
		}
	}
	return nil
}

func socialTaskMediaContentType(ref *SocialTaskMediaRef) string {
	if ref == nil {
		return ""
	}
	if contentType := strings.ToLower(strings.TrimSpace(ref.ContentType)); contentType != "" {
		return contentType
	}
	rawURL := strings.TrimSpace(ref.URL)
	if !strings.HasPrefix(strings.ToLower(rawURL), "data:") {
		return ""
	}
	comma := strings.Index(rawURL, ",")
	if comma <= len("data:") {
		return ""
	}
	meta := strings.TrimSpace(rawURL[len("data:"):comma])
	if meta == "" {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(strings.SplitN(meta, ";", 2)[0]))
}

func socialTaskMediaDimensions(ref *SocialTaskMediaRef) (width int, height int, known bool, err error) {
	if ref == nil || ref.IsZero() {
		return 0, 0, false, nil
	}

	rawURL := strings.TrimSpace(ref.URL)
	if strings.HasPrefix(strings.ToLower(rawURL), "data:image/") {
		body, err := decodeSocialTaskInlineImageDataURL(rawURL)
		if err != nil {
			return 0, 0, true, err
		}
		width, height, err = decodeSocialTaskImageDimensions(body)
		if err != nil {
			return 0, 0, true, err
		}
		return width, height, true, nil
	}

	width = ref.Width
	height = ref.Height
	if width > 0 || height > 0 {
		if width <= 0 || height <= 0 {
			return 0, 0, false, nil
		}
		return width, height, true, nil
	}
	return 0, 0, false, nil
}

func decodeSocialTaskInlineImageDataURL(raw string) ([]byte, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("data url is required")
	}
	if !strings.HasPrefix(strings.ToLower(raw), "data:image/") {
		return nil, fmt.Errorf("only inline image data urls are supported")
	}
	comma := strings.Index(raw, ",")
	if comma <= len("data:") {
		return nil, fmt.Errorf("data url is malformed")
	}
	meta := raw[len("data:"):comma]
	dataPart := strings.TrimSpace(raw[comma+1:])
	if !strings.HasSuffix(strings.ToLower(meta), ";base64") {
		return nil, fmt.Errorf("data url must be base64 encoded")
	}
	decoded, err := base64.StdEncoding.DecodeString(dataPart)
	if err != nil {
		return nil, err
	}
	if len(decoded) == 0 {
		return nil, fmt.Errorf("image body is empty")
	}
	return decoded, nil
}

func decodeSocialTaskImageDimensions(body []byte) (int, int, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		return 0, 0, err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return 0, 0, fmt.Errorf("image dimensions are invalid")
	}
	return cfg.Width, cfg.Height, nil
}
