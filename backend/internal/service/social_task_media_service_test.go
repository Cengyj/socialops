package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSocialTaskMediaFileExtensionFromContentTypeUsesCurrentImageBoundary(t *testing.T) {
	cases := map[string]string{
		"image/png":  ".png",
		"image/jpeg": ".jpg",
		"image/jpg":  ".jpg",
		"image/gif":  ".gif",
		"image/webp": ".webp",
	}

	for contentType, want := range cases {
		t.Run(contentType, func(t *testing.T) {
			require.Equal(t, want, socialTaskMediaFileExtensionFromContentType(contentType))
			require.Equal(t, want, socialTaskMediaFileExtensionFromContentType(strings.ToUpper(contentType)))
		})
	}

	require.Empty(t, socialTaskMediaFileExtensionFromContentType("video/mp4"))
	require.Empty(t, socialTaskMediaFileExtensionFromContentType("application/pdf"))
}
