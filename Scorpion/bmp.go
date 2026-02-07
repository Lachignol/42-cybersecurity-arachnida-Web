package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Bmp struct {
	width, height uint32
	bitsPerPixel  uint16
	Tags          map[string]string
}

func newBmp() *Bmp {
	return &Bmp{Tags: make(map[string]string)}
}

func clear_bmp(pathOfFile string) {
	content, err := openAndExtractContent(pathOfFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to read file: %v\n", err)
		return
	}

	if !verify_bmp(content) {
		fmt.Fprintf(os.Stderr, "Error: not a valid BMP\n")
		return
	}

	ext := filepath.Ext(pathOfFile)
	outFile := strings.TrimSuffix(pathOfFile, ext) + "_clear.bmp"

	bmp := newBmp()
	parse_bmp(content, bmp)

	cleanedSize := createMinimalBmp(outFile, bmp, content)

	metadataRemoved := len(content) - cleanedSize
	PrintCleanResult("BMP", outFile, len(content), cleanedSize, metadataRemoved)
}

func createMinimalBmp(outFile string, bmp *Bmp, original []byte) int {
	// Recup les pixel de l'image originial Le c'est pour little endian
	offset := readUint32LE(original[10:14])
	pixels := original[offset:]

	f, _ := os.Create(outFile)
	defer f.Close()

	// header BMP minimal (54 bytes)
	header := make([]byte, 54)
	copy(header[:2], []byte("BM"))                   // Signature
	binary.LittleEndian.PutUint32(header[10:14], 54) // Offset des data
	binary.LittleEndian.PutUint32(header[14:18], 40) // DIB size = 40
	binary.LittleEndian.PutUint32(header[18:22], bmp.width)
	binary.LittleEndian.PutUint32(header[22:26], bmp.height)
	binary.LittleEndian.PutUint16(header[26:28], 1) // Planes
	binary.LittleEndian.PutUint16(header[28:30], bmp.bitsPerPixel)

	// ecrure header et les pixels
	bytesWritten := 0
	n, _ := f.Write(header)
	bytesWritten += n
	n, _ = f.Write(pixels)
	bytesWritten += n

	return bytesWritten
}

func bmp(pathOfFile string) (map[string]string, error) {
	bmp := newBmp()
	content, err := openAndExtractContent(pathOfFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if !verify_bmp(content) {
		return nil, fmt.Errorf("not a valid BMP file")
	}
	parse_bmp(content, bmp)
	for k, v := range bmp.Tags {
		PrintMetadata(k, v)
	}

	return bmp.Tags, err
}

func parse_bmp(content []byte, bmp *Bmp) {
	// BMP File Header toujours 14
	if len(content) < 14 {
		return
	}
	bmp.Tags["Signature"] = string(content[0:2]) // e.g. "BM"
	bmp.Tags["FileSize"] = fmt.Sprintf("%d bytes", readUint32LE(content[2:6]))
	bmp.Tags["Reserved1"] = fmt.Sprintf("0x%04x", readUint16LE(content[6:8]))
	bmp.Tags["Reserved2"] = fmt.Sprintf("0x%04x", readUint16LE(content[8:10]))
	bmp.Tags["DataOffset"] = fmt.Sprintf("0x%x", readUint32LE(content[10:14]))

	// DIB Header size a partir de 14 + 4 bytes
	if len(content) < 18 {
		return
	}
	dibSize := readUint32LE(content[14:18])
	bmp.Tags["DIBHeaderSize"] = fmt.Sprintf("%d bytes (V%d)", dibSize, dibVersion(dibSize))
	offset := 18

	switch dibSize {
	case 12:
		if len(content) < offset+12-4 {
			return
		}
		bmp.width = uint32(readUint16LE(content[offset : offset+2]))
		bmp.height = uint32(readUint16LE(content[offset+2 : offset+4]))
		bmp.bitsPerPixel = readUint16LE(content[offset+6 : offset+8])
		bmp.Tags["Width"] = fmt.Sprintf("%d px", bmp.width)
		bmp.Tags["Height"] = fmt.Sprintf("%d px", bmp.height)
		bmp.Tags["Planes"] = fmt.Sprintf("%d", readUint16LE(content[offset+4:offset+6]))
		bmp.Tags["BitsPerPixel"] = fmt.Sprintf("%d bpp", bmp.bitsPerPixel)

	case 40, 52, 56, 108, 124:
		if len(content) < offset+int(dibSize) {
			return
		}
		bmp.width = readUint32LE(content[offset : offset+4])
		bmp.height = readUint32LE(content[offset+4 : offset+8])
		bmp.bitsPerPixel = readUint16LE(content[offset+10 : offset+12])
		bmp.Tags["Width"] = fmt.Sprintf("%d px", bmp.width)
		bmp.Tags["Height"] = fmt.Sprintf("%d px", bmp.height)
		bmp.Tags["Planes"] = fmt.Sprintf("%d", readUint16LE(content[offset+8:offset+10]))
		bmp.Tags["BitsPerPixel"] = fmt.Sprintf("%d bpp", bmp.bitsPerPixel)
		comp := readUint32LE(content[offset+12 : offset+16])
		bmp.Tags["Compression"] = compressionName(comp)
		bmp.Tags["CompressionRaw"] = fmt.Sprintf("0x%x", comp)
		bmp.Tags["ImageSize"] = fmt.Sprintf("%d bytes", readUint32LE(content[offset+16:offset+20]))
		bmp.Tags["XPixelsPerMeter"] = fmt.Sprintf("%d", readUint32LE(content[offset+20:offset+24]))
		bmp.Tags["YPixelsPerMeter"] = fmt.Sprintf("%d", readUint32LE(content[offset+24:offset+28]))
		bmp.Tags["ColorsUsed"] = fmt.Sprintf("%d", readUint32LE(content[offset+28:offset+32]))
		bmp.Tags["ColorsImportant"] = fmt.Sprintf("%d", readUint32LE(content[offset+32:offset+36]))

		advance := 36

		if dibSize >= 52 {
			bmp.Tags["RedMask"] = fmt.Sprintf("0x%08x", readUint32LE(content[offset+advance:offset+advance+4]))
			bmp.Tags["GreenMask"] = fmt.Sprintf("0x%08x", readUint32LE(content[offset+advance+4:offset+advance+8]))
			bmp.Tags["BlueMask"] = fmt.Sprintf("0x%08x", readUint32LE(content[offset+advance+8:offset+advance+12]))
			advance += 12
		}
		if dibSize >= 56 {
			bmp.Tags["AlphaMask"] = fmt.Sprintf("0x%08x", readUint32LE(content[offset+advance:offset+advance+4]))
			advance += 4
		}
		if dibSize >= 108 {
			bmp.Tags["CSType"] = fmt.Sprintf("0x%08x", readUint32LE(content[offset+advance:offset+advance+4]))
			bmp.Tags["Endpoints"] = "(36 bytes)"
			advance += 4 + 36
			bmp.Tags["GammaRed"] = fmt.Sprintf("%d", readUint32LE(content[offset+advance:offset+advance+4]))
			bmp.Tags["GammaGreen"] = fmt.Sprintf("%d", readUint32LE(content[offset+advance+4:offset+advance+8]))
			bmp.Tags["GammaBlue"] = fmt.Sprintf("%d", readUint32LE(content[offset+advance+8:offset+advance+12]))
			advance += 12
		}
		if dibSize >= 124 {
			bmp.Tags["Intent"] = fmt.Sprintf("0x%08x", readUint32LE(content[offset+advance:offset+advance+4]))
			bmp.Tags["ProfileData"] = fmt.Sprintf("%d", readUint32LE(content[offset+advance+4:offset+advance+8]))
			bmp.Tags["ProfileSize"] = fmt.Sprintf("%d", readUint32LE(content[offset+advance+8:offset+advance+12]))
			bmp.Tags["Reserved"] = fmt.Sprintf("0x%08x", readUint32LE(content[offset+advance+12:offset+advance+16]))
		}
	}
}

func verify_bmp(content []byte) bool {
	if len(content) < 54 || content[0] != 'B' || content[1] != 'M' {
		return false
	}
	return true
}

func readUint32LE(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func readUint16LE(b []byte) uint16 {
	return uint16(b[0]) | uint16(b[1])<<8
}

func dibVersion(size uint32) int {
	switch size {
	case 12:
		return 1 // OS/2 1.x
	case 40:
		return 3 // Windows 3.x
	case 64:
		return 2 // OS/2 2.x
	case 108:
		return 4 // Windows 4.x
	case 124:
		return 5 // Windows 5.x
	default:
		return 0
	}
}

func compressionName(comp uint32) string {
	switch comp {
	case 0:
		return "None"
	case 1:
		return "8bit RLE"
	case 2:
		return "4bit RLE"
	case 3:
		return "Bitfields"
	case 4:
		return "JPEG"
	case 5:
		return "PNG"
	default:
		return fmt.Sprintf("Unknown(%d)", comp)
	}
}
