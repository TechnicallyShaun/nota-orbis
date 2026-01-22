package metadata

import (
	"encoding/binary"
	"os"
	"time"
)

// createTestM4A creates a minimal valid M4A file for testing.
// The file contains ftyp, moov/mvhd boxes with creation time and duration.
func createTestM4A(path string, creationTime time.Time, durationSeconds uint32) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// ftyp box (file type)
	ftyp := []byte{
		0x00, 0x00, 0x00, 0x14, // size: 20 bytes
		'f', 't', 'y', 'p',
		'M', '4', 'A', ' ', // major brand
		0x00, 0x00, 0x00, 0x00, // minor version
		'M', '4', 'A', ' ', // compatible brand
	}
	if _, err := f.Write(ftyp); err != nil {
		return err
	}

	// Convert time to Mac epoch (seconds since 1904-01-01)
	macEpoch := time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
	macTime := uint32(creationTime.Sub(macEpoch).Seconds())

	// mvhd box (movie header) - version 0
	mvhdData := make([]byte, 108)
	mvhdData[0] = 0 // version
	// flags: bytes 1-3 are 0
	binary.BigEndian.PutUint32(mvhdData[4:8], macTime)   // creation time
	binary.BigEndian.PutUint32(mvhdData[8:12], macTime)  // modification time
	binary.BigEndian.PutUint32(mvhdData[12:16], 1000)    // timescale (1000 = milliseconds)
	binary.BigEndian.PutUint32(mvhdData[16:20], durationSeconds*1000) // duration in timescale units
	binary.BigEndian.PutUint32(mvhdData[20:24], 0x00010000) // rate (1.0)
	binary.BigEndian.PutUint16(mvhdData[24:26], 0x0100)     // volume (1.0)
	// rest is padding and matrix

	mvhdBox := make([]byte, 8+108)
	binary.BigEndian.PutUint32(mvhdBox[0:4], 116) // size
	copy(mvhdBox[4:8], []byte("mvhd"))
	copy(mvhdBox[8:], mvhdData)

	// moov box (movie)
	moovSize := uint32(8 + len(mvhdBox))
	moovHeader := make([]byte, 8)
	binary.BigEndian.PutUint32(moovHeader[0:4], moovSize)
	copy(moovHeader[4:8], []byte("moov"))

	if _, err := f.Write(moovHeader); err != nil {
		return err
	}
	if _, err := f.Write(mvhdBox); err != nil {
		return err
	}

	return nil
}

// createInvalidM4A creates a file that is not a valid M4A.
func createInvalidM4A(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write some garbage that looks like a box header but isn't M4A
	ftyp := []byte{
		0x00, 0x00, 0x00, 0x14, // size: 20 bytes
		'f', 't', 'y', 'p',
		'X', 'X', 'X', 'X', // invalid brand
		0x00, 0x00, 0x00, 0x00,
		'X', 'X', 'X', 'X',
	}
	_, err = f.Write(ftyp)
	return err
}
