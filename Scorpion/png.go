package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

type Png struct {
	width, height                  uint32
	bitDepth, colorType            byte
	compression, filter, interlace byte
	Tags                           map[string]string
	chunks                         map[string][]byte
}

func newPng() *Png {
	return &Png{
		Tags:   make(map[string]string),
		chunks: make(map[string][]byte),
	}
}

func clear_png(pathOfFile string) {
	content, err := openAndExtractContent(pathOfFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to read file: %v\n", err)
		return
	}
	if !verify_png(content) {
		fmt.Fprintf(os.Stderr, "Error: not a valid PNG file\n")
		return
	}

	// je recup le nom du nouveau fichier cree et le nombre de byte que j'ai ecrit
	cleanName, cleanedSize := removePNGMetadata(pathOfFile, content)
	if cleanName == "" {
		fmt.Fprintf(os.Stderr, "Error: failed to remove metadata\n")
		return
	}
	// je print la diff entre avant et apres pour voir ce que jai cleaner
	metadataRemoved := len(content) - cleanedSize
	PrintCleanResult("PNG", cleanName, len(content), cleanedSize, metadataRemoved)
}

func removePNGMetadata(pathOfFile string, content []byte) (string, int) {
	ext := filepath.Ext(pathOfFile)
	nameOfNewFile := strings.TrimSuffix(pathOfFile, ext)
	nameOfNewFile = nameOfNewFile + "clear" + ext
	f, err := os.Create(nameOfNewFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: can't create file %s: %v\n", nameOfNewFile, err)
		return "", 0
	}
	defer f.Close()

	// j'ecri le header PNG obliger
	header := content[:8]
	bytesWritten, _ := f.Write(header)
	offset := 8

	// la je boucle sur tout les chunk
	for offset+12 <= len(content) {
		length := int(content[offset])<<24 | int(content[offset+1])<<16 | int(content[offset+2])<<8 | int(content[offset+3])

		chunkType := string(content[offset+4 : offset+8])
		totalChunkLen := 12 + length // 4 = len  + 4 = type + data + 4 = crc (en gros signature de  fin de chunk)
		if offset+totalChunkLen > len(content) {
			break
		}
		chunkData := content[offset : offset+totalChunkLen]
		// je filtre sur ce qui est essentiel pour pas casser l'image
		switch chunkType {
		case "IHDR", "IDAT", "IEND":
			n, _ := f.Write(chunkData)
			bytesWritten += n
		default:
			// si tEXt, zTXt, iTXt, tIME, pHYs, j'ecri pas j'ignore.
		}

		offset += totalChunkLen
	}

	return nameOfNewFile, bytesWritten
}

func png(pathOfFile string) (map[string]string, error) {
	png := newPng()
	content, err := openAndExtractContent(pathOfFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	if !verify_png(content) {
		return nil, fmt.Errorf("not a valid PNG file")
	}
	parsePngChunks(content[8:], png) // je lui envoi direct apres la signature png
	names := make([]string, 0, len(png.Tags))
	for k := range png.Tags {
		names = append(names, k)
	}
	// je trie un peu les tags
	slices.Sort(names)
	for _, k := range names {
		PrintMetadata(k, png.Tags[k])
	}
	//print general a voir si je laisse
	PrintImageInfo(fmt.Sprintf("PNG %dx%d bitDepth=%d colorType=%d",
		png.width, png.height, png.bitDepth, png.colorType))

	return png.Tags, nil
}

func verify_png(content []byte) bool {
	// c'est la signature de tout les png
	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if len(content) < 8 || !bytes.Equal(content[:8], pngSignature) {
		return false
	}
	return true
}

func parsePngChunks(data []byte, png *Png) {
	i := 0

	for i+12 <= len(data) {
		length := int(data[i])<<24 | int(data[i+1])<<16 |
			int(data[i+2])<<8 | int(data[i+3])
		chunkType := string(data[i+4 : i+8])
		chunkData := data[i+8 : i+8+length]
		png.chunks[chunkType] = chunkData
		switch chunkType {
		case "IHDR":
			parseIHDR(chunkData, png)
		case "PLTE":
			png.Tags["PaletteSize"] = fmt.Sprintf("%d colors", len(chunkData)/3)
		case "gAMA":
			gamma := float64(binary.BigEndian.Uint32(chunkData)) / 100000.0
			png.Tags["Gamma"] = fmt.Sprintf("%.4f", gamma)

		case "sRGB":
			if len(chunkData) == 1 {
				intents := []string{"Perceptual", "RelativeColorimetric", "Saturation", "AbsoluteColorimetric"}
				png.Tags["sRGBRenderingIntent"] = intents[chunkData[0]]
			}
		case "cHRM":
			parseChrmChunk(chunkData, png)
		case "tRNS":
			parseTRNS(chunkData, png)
		case "pHYs":
			parsePhysChunk(chunkData, png)
		case "bKGD":
			parseBKGD(chunkData, png)
		case "hIST":
			png.Tags["HistogramEntries"] = fmt.Sprintf("%d", len(chunkData)/2)
		case "tIME":
			parseTimeChunk(chunkData, png)
		case "iCCP":
			parseICCProfile(chunkData, png)
		case "tEXt", "iTXt", "zTXt":
			parsePngText(chunkData, chunkType, png)
		}
		// j'increment de la taille du chunk donc 12 octet + la taille de la data que jai recup dans length
		i += 12 + length
	}
	computePngFields(png)
}

func parseTRNS(data []byte, png *Png) {
	switch png.colorType {
	case 0: // Grayscale
		if len(data) >= 2 {
			png.Tags["tRNSGray"] = fmt.Sprintf("%d", binary.BigEndian.Uint16(data[:2]))
		}
	case 2: // Truecolor
		if len(data) >= 6 {
			png.Tags["tRNSRed"] = fmt.Sprintf("%d", binary.BigEndian.Uint16(data[0:2]))
			png.Tags["tRNSGreen"] = fmt.Sprintf("%d", binary.BigEndian.Uint16(data[2:4]))
			png.Tags["tRNSBlue"] = fmt.Sprintf("%d", binary.BigEndian.Uint16(data[4:6]))
		}
	case 3: // Indexed
		png.Tags["tRNSPaletteEntries"] = fmt.Sprintf("%d", len(data))
	case 4: // Grayscale+Alpha
		png.Tags["tRNSGrayAlpha"] = fmt.Sprintf("%d", data[0])
	}
}

func parseBKGD(data []byte, png *Png) {
	switch png.colorType {
	case 0, 4: // Grayscale
		png.Tags["BackgroundGray"] = fmt.Sprintf("%d", data[0])
	case 2: // RGB
		png.Tags["BackgroundRGB"] = fmt.Sprintf("%02x%02x%02x", data[0], data[1], data[2])
	case 3: // Palette
		png.Tags["BackgroundPaletteIndex"] = fmt.Sprintf("%d", data[0])
	}
}

func parseICCProfile(data []byte, png *Png) {
	nulIdx := bytes.IndexByte(data, 0)
	if nulIdx <= 0 {
		return
	}
	png.Tags["ProfileName"] = string(data[:nulIdx])
	if nulIdx+2 < len(data) && data[nulIdx+1] == 0 {
		r, err := zlib.NewReader(bytes.NewReader(data[nulIdx+2:]))
		if err != nil {
			return
		}
		defer r.Close()
		iccData, err := io.ReadAll(r)
		if err != nil {
			return
		}
		parseICCData(iccData, png)
	}
}

func parseICCData(iccData []byte, png *Png) {
	if len(iccData) < 132 {
		return
	}

	// list des offset icc
	iccFields := map[int]string{
		4:  "ProfileCMMType",         // bytes 4-7
		8:  "ProfileVersion",         // bytes 8-9
		12: "ProfileClass",           // bytes 12-15
		16: "ColorSpaceData",         // bytes 16-19
		20: "ProfileConnectionSpace", // bytes 20-23
		36: "ProfileFileSignature",   // bytes 36-39
		44: "PrimaryPlatform",        // bytes 44-47
		52: "ProfileCreator",         // bytes 52-55
		72: "DeviceManufacturer",     // bytes 72-75
		76: "DeviceModel",            // bytes 76-83
	}

	for offset, fieldName := range iccFields {
		if offset+4 <= len(iccData) {
			rawBytes := iccData[offset : offset+4]
			// rappel de la piscine je boucle et rentre dans str que si c'est char imprimable
			str := ""
			for _, b := range rawBytes {
				if b >= 32 && b < 127 {
					str += string(b)
				}
			}
			str = strings.TrimSpace(str)
			// j'ajoute que si pas vide
			if str != "" {
				png.Tags[fieldName] = str
			}
		}
	}

	if len(iccData) >= 10 {
		version := binary.BigEndian.Uint16(iccData[8:10])
		png.Tags["ProfileVersion"] = fmt.Sprintf("%d.%d.%d",
			byte(version>>8), byte(version>>4&0xF), byte(version&0xF))
	}
	if len(iccData) >= 36 {
		year := binary.BigEndian.Uint16(iccData[24:26])
		month := binary.BigEndian.Uint16(iccData[26:28])
		day := binary.BigEndian.Uint16(iccData[28:30])
		hour := binary.BigEndian.Uint16(iccData[30:32])
		min := binary.BigEndian.Uint16(iccData[32:34])
		sec := binary.BigEndian.Uint16(iccData[34:36])
		png.Tags["ProfileDateTime"] = fmt.Sprintf("%04d:%02d:%02d %02d:%02d:%02d",
			year, month, day, hour, min, sec)
	}

	if len(iccData) >= 92 {
		id := binary.BigEndian.Uint64(iccData[84:92])
		png.Tags["ProfileID"] = fmt.Sprintf("%d", id)
	}
}
func parseChrmChunk(data []byte, png *Png) {
	if len(data) != 32 {
		return
	}
	div := 100000.0

	offsets := []struct {
		name  string
		start int
	}{
		{"MediaWhitePoint", 0},
		{"RedMatrixColumn", 8},
		{"GreenMatrixColumn", 16},
		{"BlueMatrixColumn", 24},
	}

	for _, f := range offsets {
		if f.start+8 <= len(data) {
			x := float64(binary.BigEndian.Uint32(data[f.start:f.start+4])) / div
			y := float64(binary.BigEndian.Uint32(data[f.start+4:f.start+8])) / div
			png.Tags[f.name] = fmt.Sprintf("%.5f %.5f", x, y)
		}
	}
}
func computePngFields(png *Png) {
	colorTypes := map[byte]string{
		0: "Grayscale", 2: "RGB", 3: "Indexed",
		4: "Grayscale+Alpha", 6: "RGB with Alpha",
	}
	if name, ok := colorTypes[png.colorType]; ok {
		png.Tags["ColorType"] = name
	}

	png.Tags["BitDepth"] = fmt.Sprintf("%d", png.bitDepth)
	png.Tags["ImageSize"] = fmt.Sprintf("%dx%d", png.width, png.height)
	png.Tags["Megapixels"] = fmt.Sprintf("%.1f", float64(png.width*png.height)/1e6)

	// PNG (toujours 0) donc generique
	png.Tags["Compression"] = "Deflate/Inflate"
	png.Tags["Filter"] = "Adaptive"
	png.Tags["Interlace"] = "Noninterlaced"
}

func parseIHDR(data []byte, png *Png) {
	if len(data) < 13 {
		return
	}
	png.width = binary.BigEndian.Uint32(data[0:4])
	png.height = binary.BigEndian.Uint32(data[4:8])
	png.bitDepth = data[8]
	png.colorType = data[9]
	png.compression = data[10]
	png.filter = data[11]
	png.interlace = data[12]
}

func isBinary(s string) bool {
	b := []byte(s)
	for _, c := range b {
		if c == 0x00 || c < 0x20 && c != 0x09 && c != 0x0A && c != 0x0D {
			return true
		}
	}
	return false
}

func parsePngText(data []byte, chunkType string, png *Png) {
	switch chunkType {

	case "tEXt":
		nulIdx := bytes.IndexByte(data, 0)
		if nulIdx > 0 && nulIdx < len(data)-1 {
			keyword := string(data[:nulIdx])
			value := string(data[nulIdx+1:])

			if isBinary(value) {
				png.Tags[keyword] = fmt.Sprintf("[Binary %d bytes]", len(value))
			} else {
				png.Tags[keyword] = value
			}
		}
	case "iTXt":
		parts := bytes.SplitN(data, []byte{0}, 6)
		if len(parts) >= 6 {
			keyword := string(parts[0])
			if strings.HasPrefix(keyword, "XML:") {
				parseXMP(parts[5], png)
			} else {
				value := string(parts[5])
				png.Tags[keyword] = value
			}
		}
	case "zTXt":
		nulIdx := bytes.IndexByte(data, 0)
		if nulIdx > 0 && nulIdx < len(data)-2 {

			keyword := string(data[:nulIdx])
			// data[nulIdx] = 0x00 (séparateur)
			// data[nulIdx+1] = méthode de compression (doit être 0 pour zlib)
			// Le reste c'est ce qui est compresser'
			compressedData := bytes.NewReader(data[nulIdx+2:])
			r, err := zlib.NewReader(compressedData)
			if err == nil {
				decompressed, err := io.ReadAll(r)
				r.Close()
				if err == nil {
					png.Tags[keyword] = string(decompressed)
					// Si c'est du XMP compressé, on le parse aussi
					if keyword == "XML:com.adobe.xmp" {
						parseXMP(decompressed, png)
					}
				} else {
					png.Tags[keyword] = "[zlib error]"
				}
			}
		}
		// comme tEXt mais compressé zlib je gere pas je di juste que ca existe
		// nulIdx := bytes.IndexByte(data, 0)
		// if nulIdx > 0 {
		// 	keyword := string(data[:nulIdx])
		// 	png.Tags[keyword] = "[zTXt compressed]"
		// }
	}

}

func parseXMP(xmlData []byte, png *Png) {
	xmp := string(xmlData)

	reXDim := regexp.MustCompile(`exif:PixelXDimension>(\d+)<`)
	reYDim := regexp.MustCompile(`exif:PixelYDimension>(\d+)<`)
	reOrient := regexp.MustCompile(`tiff:Orientation>(\d+)<`)

	if match := reXDim.FindStringSubmatch(xmp); len(match) > 1 {
		png.Tags["XMP:PixelXDimension"] = match[1]
	}
	if match := reYDim.FindStringSubmatch(xmp); len(match) > 1 {
		png.Tags["XMP:PixelYDimension"] = match[1]
	}
	if match := reOrient.FindStringSubmatch(xmp); len(match) > 1 {
		png.Tags["XMP:Orientation"] = match[1]
	}
	reTitle := regexp.MustCompile(`dc:title[^>]*>([^<]+)<`)
	reCreator := regexp.MustCompile(`dc:creator[^>]*>([^<]+)<`)
	reDescription := regexp.MustCompile(`dc:description[^>]*>([^<]+)<`)
	if match := reTitle.FindStringSubmatch(xmp); len(match) > 1 {
		png.Tags["XMP:Title"] = strings.TrimSpace(match[1])
	}
	if match := reCreator.FindStringSubmatch(xmp); len(match) > 1 {
		png.Tags["XMP:Creator"] = strings.TrimSpace(match[1])
	}
	if match := reDescription.FindStringSubmatch(xmp); len(match) > 1 {
		png.Tags["XMP:Description"] = strings.TrimSpace(match[1])
	}
}

func parseTimeChunk(data []byte, png *Png) {
	if len(data) == 7 {
		year := binary.BigEndian.Uint16(data[0:2])
		month := data[2]
		day := data[3]
		hour := data[4]
		min := data[5]
		sec := data[6]
		png.Tags["ModificationTime"] = fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d",
			year, month, day, hour, min, sec)
	}
}

func parsePhysChunk(data []byte, png *Png) {
	if len(data) == 9 {
		xppu := binary.BigEndian.Uint32(data[0:4])
		yppu := binary.BigEndian.Uint32(data[4:8])
		unit := data[8]
		png.Tags["PixelsPerUnitX"] = fmt.Sprintf("%d", xppu)
		png.Tags["PixelsPerUnitY"] = fmt.Sprintf("%d", yppu)
		unitNames := map[byte]string{0: "meters", 1: "pixels"}
		if name, ok := unitNames[unit]; ok {
			png.Tags["PixelUnits"] = name
		} else {
			png.Tags["PixelUnits"] = fmt.Sprintf("unit=%d", unit)
		}
	}
}
