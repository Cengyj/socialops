package service

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"image"
	"image/color"
	stddraw "image/draw"
	"image/jpeg"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"

	_ "image/gif"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

const socialTaskNormalizedImageQuality = 90

func normalizeTwitterProfileMedia(media *twitterPreparedMedia, label string, targetWidth, targetHeight int) (*twitterPreparedMedia, error) {
	if media == nil || len(media.body) == 0 {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, fmt.Sprintf("%s media is required", label), nil)
	}

	width, height, err := decodeSocialTaskImageDimensions(media.body)
	if err != nil || width <= 0 || height <= 0 {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, fmt.Sprintf("%s media is invalid", label), err)
	}
	if width == targetWidth && height == targetHeight {
		return media, nil
	}

	decoded, _, err := image.Decode(bytes.NewReader(media.body))
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, fmt.Sprintf("%s media is invalid", label), err)
	}
	if decoded.Bounds().Empty() {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, fmt.Sprintf("%s media is invalid", label), nil)
	}

	normalizedBody, err := resizeSocialTaskImageToTarget(decoded, targetWidth, targetHeight)
	if err != nil {
		return nil, newSocialExecutionError(SocialExecutionFailureActionInput, fmt.Sprintf("%s media is invalid", label), err)
	}
	fileName := normalizedSocialTaskImageFileName(media.fileName, label)
	md5Sum := md5.Sum(normalizedBody)
	return &twitterPreparedMedia{
		fieldName:   media.fieldName,
		contentType: "image/jpeg",
		fileName:    fileName,
		body:        normalizedBody,
		md5Hex:      fmt.Sprintf("%x", md5Sum[:]),
	}, nil
}

func resizeSocialTaskImageToTarget(src image.Image, targetWidth, targetHeight int) ([]byte, error) {
	if src == nil {
		return nil, fmt.Errorf("image is required")
	}
	if targetWidth <= 0 || targetHeight <= 0 {
		return nil, fmt.Errorf("target size is invalid")
	}

	srcBounds := src.Bounds()
	if srcBounds.Empty() {
		return nil, fmt.Errorf("image bounds are empty")
	}

	crop := cropSocialTaskImageRect(srcBounds, targetWidth, targetHeight)
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	stddraw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, stddraw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, crop, stddraw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: socialTaskNormalizedImageQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func cropSocialTaskImageRect(bounds image.Rectangle, targetWidth, targetHeight int) image.Rectangle {
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()
	targetRatio := float64(targetWidth) / float64(targetHeight)
	srcRatio := float64(srcWidth) / float64(srcHeight)

	if srcRatio > targetRatio {
		cropWidth := int(float64(srcHeight) * targetRatio)
		if cropWidth < 1 {
			cropWidth = 1
		}
		if cropWidth > srcWidth {
			cropWidth = srcWidth
		}
		offsetX := (srcWidth - cropWidth) / 2
		return image.Rect(bounds.Min.X+offsetX, bounds.Min.Y, bounds.Min.X+offsetX+cropWidth, bounds.Max.Y)
	}
	if srcRatio < targetRatio {
		cropHeight := int(float64(srcWidth) / targetRatio)
		if cropHeight < 1 {
			cropHeight = 1
		}
		if cropHeight > srcHeight {
			cropHeight = srcHeight
		}
		offsetY := (srcHeight - cropHeight) / 2
		return image.Rect(bounds.Min.X, bounds.Min.Y+offsetY, bounds.Max.X, bounds.Min.Y+offsetY+cropHeight)
	}
	return bounds
}

func normalizedSocialTaskImageFileName(fileName, fallback string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return fallback + ".jpg"
	}
	ext := filepath.Ext(fileName)
	base := strings.TrimSuffix(fileName, ext)
	if strings.TrimSpace(base) == "" {
		base = fallback
	}
	return base + ".jpg"
}
