package socialidentity

import (
	"strings"
	"unicode"
)

const (
	KindUsername = "username"
)

type BusinessIdentity struct {
	PlatformKey  string
	NameKey      string
	IdentityKind string
	IdentityKey  string
}

func NormalizePlatform(platform string) string {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(platform)), func(r rune) bool {
		return r == '-' || r == '/' || r == '_' || unicode.IsSpace(r)
	})
	normalized := strings.Join(parts, "_")
	switch normalized {
	case "x", "twitter", "x_twitter", "twitter_x":
		return "x_twitter"
	default:
		return normalized
	}
}

func NormalizeUsername(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	normalized = strings.TrimLeft(normalized, "@")
	return strings.TrimSpace(normalized)
}

func Build(platform, name string) BusinessIdentity {
	platformKey := NormalizePlatform(platform)
	nameKey := NormalizeUsername(name)
	return BusinessIdentity{
		PlatformKey:  platformKey,
		NameKey:      nameKey,
		IdentityKind: KindUsername,
		IdentityKey:  nameKey,
	}
}

func DedupKey(platform, name string) string {
	identity := Build(platform, name)
	if identity.PlatformKey == "" || identity.IdentityKey == "" {
		return ""
	}
	return identity.PlatformKey + "\x00" + identity.IdentityKind + "\x00" + identity.IdentityKey
}
