package crypto

import (
	"fmt"
	"strings"
)

type Profile string

const (
	ProfileModern Profile = "modern"
	ProfileNIST   Profile = "nist"
)

func ParseProfile(value string) (Profile, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(ProfileModern):
		return ProfileModern, nil
	case string(ProfileNIST):
		return ProfileNIST, nil
	default:
		return "", fmt.Errorf("unsupported crypto profile %q", value)
	}
}
