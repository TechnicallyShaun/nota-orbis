// Package metadata provides audio file metadata extraction for the transcription service.
package metadata

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"time"
)

// ErrInvalidFormat indicates the file is not a valid M4A/MP4 file.
var ErrInvalidFormat = errors.New("invalid M4A format")

// AudioMetadata contains extracted metadata from an audio file.
type AudioMetadata struct {
	CreationTime time.Time
	Duration     time.Duration
	Title        string
}

// ExtractM4A extracts metadata from an M4A file.
func ExtractM4A(path string) (*AudioMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseM4A(f)
}

func parseM4A(r io.ReadSeeker) (*AudioMetadata, error) {
	meta := &AudioMetadata{}
	var foundFtyp, foundMoov bool

	// M4A files are based on the ISO base media file format (MP4)
	// They consist of boxes (atoms) with a size and type
	for {
		boxSize, boxType, err := readBoxHeader(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch boxType {
		case "moov":
			// Movie box contains metadata - descend into it
			if err := parseMoov(r, boxSize-8, meta); err != nil {
				return nil, err
			}
			foundMoov = true
		case "ftyp":
			// File type box - validate it's an M4A compatible format
			if err := validateFtyp(r, boxSize-8); err != nil {
				return nil, err
			}
			foundFtyp = true
		default:
			// Skip unknown boxes
			if _, err := r.Seek(int64(boxSize-8), io.SeekCurrent); err != nil {
				return nil, err
			}
		}
	}

	if !foundFtyp || !foundMoov {
		return nil, ErrInvalidFormat
	}

	return meta, nil
}

func readBoxHeader(r io.Reader) (uint32, string, error) {
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return 0, "", err
	}

	size := binary.BigEndian.Uint32(header[0:4])
	boxType := string(header[4:8])

	return size, boxType, nil
}

func validateFtyp(r io.ReadSeeker, remaining uint32) error {
	brand := make([]byte, 4)
	if _, err := io.ReadFull(r, brand); err != nil {
		return err
	}

	// Check for M4A compatible brands
	brandStr := string(brand)
	validBrands := []string{"M4A ", "mp41", "mp42", "isom"}
	valid := false
	for _, vb := range validBrands {
		if brandStr == vb {
			valid = true
			break
		}
	}

	if !valid {
		return ErrInvalidFormat
	}

	// Skip the rest of ftyp
	if remaining > 4 {
		if _, err := r.Seek(int64(remaining-4), io.SeekCurrent); err != nil {
			return err
		}
	}

	return nil
}

func parseMoov(r io.ReadSeeker, remaining uint32, meta *AudioMetadata) error {
	endPos, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	endPos += int64(remaining)

	for {
		currentPos, err := r.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}
		if currentPos >= endPos {
			break
		}

		boxSize, boxType, err := readBoxHeader(r)
		if err != nil {
			return err
		}

		switch boxType {
		case "mvhd":
			// Movie header - contains creation time and duration
			if err := parseMvhd(r, boxSize-8, meta); err != nil {
				return err
			}
		case "udta":
			// User data - may contain title
			if err := parseUdta(r, boxSize-8, meta); err != nil {
				return err
			}
		default:
			// Skip unknown boxes
			if _, err := r.Seek(int64(boxSize-8), io.SeekCurrent); err != nil {
				return err
			}
		}
	}

	return nil
}

func parseMvhd(r io.ReadSeeker, remaining uint32, meta *AudioMetadata) error {
	// Version (1 byte) + flags (3 bytes)
	var versionFlags [4]byte
	if _, err := io.ReadFull(r, versionFlags[:]); err != nil {
		return err
	}

	version := versionFlags[0]

	if version == 0 {
		// 32-bit times
		var times [8]byte
		if _, err := io.ReadFull(r, times[:]); err != nil {
			return err
		}
		creationTime := binary.BigEndian.Uint32(times[0:4])
		// Modification time at times[4:8], not needed

		// Convert from Mac epoch (1904-01-01) to Unix epoch
		macEpoch := time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
		meta.CreationTime = macEpoch.Add(time.Duration(creationTime) * time.Second)

		// Read timescale and duration (immediately after times)
		var timescaleDuration [8]byte
		if _, err := io.ReadFull(r, timescaleDuration[:]); err != nil {
			return err
		}
		timescale := binary.BigEndian.Uint32(timescaleDuration[0:4])
		duration := binary.BigEndian.Uint32(timescaleDuration[4:8])

		if timescale > 0 {
			meta.Duration = time.Duration(duration) * time.Second / time.Duration(timescale)
		}

		// Skip remaining bytes (version/flags=4 + times=8 + timescale/duration=8 = 20 bytes read)
		if remaining > 20 {
			if _, err := r.Seek(int64(remaining-20), io.SeekCurrent); err != nil {
				return err
			}
		}
	} else {
		// Version 1: 64-bit times - just skip for now
		if _, err := r.Seek(int64(remaining-4), io.SeekCurrent); err != nil {
			return err
		}
	}

	return nil
}

func parseUdta(r io.ReadSeeker, remaining uint32, meta *AudioMetadata) error {
	// User data box parsing for title - simplified implementation
	// Just skip it for now, can be enhanced later
	if _, err := r.Seek(int64(remaining), io.SeekCurrent); err != nil {
		return err
	}
	return nil
}
