package externalstorage

import "fmt"

type StorageType string

const (
	StorageNone StorageType = ""
	StorageS3   StorageType = "s3"
)

var codecs = map[StorageType]struct{}{
	StorageNone: {},
	StorageS3:   {},
}

func ParseType(t string) (StorageType, error) {
	normalised := StorageType(t)

	if _, ok := codecs[normalised]; ok {
		return normalised, nil
	}

	return "", fmt.Errorf(
		"invalid storage type %q (must be %q or %q)",
		t,
		StorageNone,
		StorageS3,
	)
}
