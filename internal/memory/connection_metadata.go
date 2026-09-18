package memory

import (
	"errors"
	"strings"
)

// ValidateConnectionMetadata checks only the supplied manifest, enrolled device
// and authorship. It shares Open/OpenService's field rules without opening a
// store or inspecting remembered content. Callers own bounded decoding, selected
// paths and the binding-to-manifest ID comparison. Success is not store health.
func ValidateConnectionMetadata(s Signet, d Device, a Authorship) error {
	if err := validateSignetMetadata(s); err != nil {
		return err
	}
	return validateAuthorship(a, func(id string) error {
		return validateDeviceMetadata(d, id)
	})
}

func validateSignetMetadata(s Signet) error {
	if s.Version != FormatVersion && s.Version != 2 {
		return errors.New("unsupported signet format")
	}
	if !identifier.MatchString(s.ID) || !textWithin(s.Name, 256) || strings.ContainsAny(s.Name, "\r\n") {
		return errors.New("invalid signet identity")
	}
	return nil
}

func validateDeviceMetadata(d Device, expectedID string) error {
	if err := ValidateDeviceID(expectedID); err != nil {
		return err
	}
	if d.ID != expectedID || d.Version != 1 || strings.TrimSpace(d.Label) == "" {
		return errors.New("invalid device identity")
	}
	return nil
}

// ValidateDeviceID permits a device identifier to be used as one metadata file
// component. It does not inspect enrollment or the filesystem.
func ValidateDeviceID(id string) error {
	if !identifier.MatchString(id) {
		return errors.New("invalid device ID")
	}
	return nil
}
