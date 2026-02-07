package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Jpeg struct {
	startMarkerExist  bool
	SOI               int
	exifSize          int
	exifStartMarker   byte
	exifStartPosition int
	isLittleEndian    bool
	marker            map[uint16]string
	byteAlign         map[byte]string
	Tags              map[string]string
}

func newJpeg() *Jpeg {

	jpeg := Jpeg{}
	jpeg.marker = map[uint16]string{
		0x010e: "ImageDescription",
		0x010f: "Make",
		0x0110: "Model",
		0x0112: "Orientation",
		0x011a: "XResolution",
		0x011b: "YResolution",
		0x0128: "ResolutionUnit",
		0x0131: "Software",
		0x0132: "DateTime",
		0x013b: "Artist",
		0x0213: "YCbCrPositioning",
		0x8769: "ExifIFDPointer",
		0x8825: "GPSInfoIFDPointer",
		0x829a: "ExposureTime",
		0x829d: "FNumber",
		0x8822: "ExposureProgram",
		0x8827: "ISOSpeedRatings",
		0x9000: "ExifVersion",
		0x9003: "DateTimeOriginal",
		0x9004: "DateTimeDigitized",
		0x9201: "ShutterSpeedValue",
		0x9202: "ApertureValue",
		0x9204: "ExposureBiasValue",
		0x9205: "MaxApertureValue",
		0x9207: "MeteringMode",
		0x9208: "LightSource",
		0x9209: "Flash",
		0x920a: "FocalLength",
		0x927c: "MakerNote",
		0x9286: "UserComment",
		0xa000: "FlashpixVersion",
		0xa001: "ColorSpace",
		0xa002: "PixelXDimension",
		0xa003: "PixelYDimension",
		0xa20e: "FocalPlaneXResolution",
		0xa20f: "FocalPlaneYResolution",
		0xa210: "FocalPlaneResolutionUnit",
		0xa401: "CustomRendered",
		0xa402: "ExposureMode",
		0xa403: "WhiteBalance",
		0xa406: "SceneCaptureType",
		0x0000: "GPSVersionID",
		0x0001: "GPSLatitudeRef",
		0x0002: "GPSLatitude",
		0x0003: "GPSLongitudeRef",
		0x0004: "GPSLongitude",
		0x0005: "GPSAltitudeRef",
		0x0006: "GPSAltitude",
		0x0010: "GPSImgDirectionRef",
		0x0011: "GPSImgDirection",
		0x001d: "GPSDateStamp",
		0xA005: "InteropIFDPointer",
		0xA420: "ImageUniqueID",
		0xC612: "DNGVersion",
		0xC61D: "ProfileCalibrationSignature",
		0x8298: "Copyright",
		0x83E7: "USITable",
		0x9010: "OffsetTime",
		0x9011: "OffsetTimeOriginal",
		0x9012: "OffsetTimeDigitized",
		0x9102: "CompressedBitsPerPixel",
		0x0102: "Compression",
		0x0103: "PhotometricInterpretation",
		0x0111: "StripOffsets",
		0x9101: "ComponentsConfiguration",
		0x9203: "BrightnessValue",
	}
	jpeg.exifStartMarker = 0xE1
	jpeg.Tags = make(map[string]string)

	return &jpeg
}

func clear_jpg(pathOfFile string) {
	content, err := openAndExtractContent(pathOfFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s: %v\n", pathOfFile, err)
		return
	}

	// Vérifier la signature JPEG
	if len(content) < 4 || content[0] != 0xFF || content[1] != 0xD8 {
		fmt.Fprintf(os.Stderr, "Error: %s: not a valid JPEG\n", pathOfFile)
		return
	}

	cleanedData, metadataRemoved := cleanAllJPEGSegments(content)

	ext := filepath.Ext(pathOfFile)
	cleanName := strings.TrimSuffix(pathOfFile, ext) + "_clear" + ext

	f, err := os.Create(cleanName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: création %s: %v\n", cleanName, err)
		return
	}
	defer f.Close()

	// Écrire le fichier nettoyé
	if _, err := f.Write(cleanedData); err != nil {
		fmt.Fprintf(os.Stderr, "Error: écriture fichier: %v\n", err)
		return
	}

	// Afficher le résultat de manière uniforme
	PrintCleanResult("JPEG", cleanName, len(content), len(cleanedData), metadataRemoved)
}

func cleanAllJPEGSegments(content []byte) ([]byte, int) {
	var cleaned []byte
	metadataBytes := 0

	// j'ecrit le SOI  = FF D8 (obligatoire)
	cleaned = append(cleaned, 0xFF, 0xD8)
	i := 2

	for i < len(content)-1 {
		if content[i] != 0xFF {
			// si pas de flag marker je 'ecri car donne brut
			cleaned = append(cleaned, content[i])
			i++
			continue
		}

		marker := content[i+1]

		// si apres markeur xFF c'est pas a prendre en compte
		if marker == 0xFF {
			cleaned = append(cleaned, content[i])
			i++
			continue
		}

		// si apres markeur x00 c'est pas a prendre en compte c'est de la  data
		if marker == 0x00 {
			cleaned = append(cleaned, content[i], content[i+1])
			i += 2
			continue
		}

		// si c'est un marker sans longeur
		if isStandaloneMarker(marker) {
			cleaned = append(cleaned, content[i], content[i+1])
			i += 2
			continue
		}

		// securite qu'on est de quoi lire apres le marker
		if i+3 >= len(content) {
			break
		}

		length := int(content[i+2])<<8 | int(content[i+3])

		// securite veirife que le segment est complet
		if i+2+length > len(content) {
			break
		}

		// voir si je garde ou non ce segment
		if shouldKeepSegment(marker) {
			// si oui je copie le segment complet
			segmentEnd := i + 2 + length
			cleaned = append(cleaned, content[i:segmentEnd]...)
		} else {
			// si on garde pas j'ecrit pas et je  compte les bytes supp
			metadataBytes += 2 + length // marker + longueur + data
		}

		i += 2 + length
	}

	// Ecrire EOI (FF D9) en gros la fin obigatoire d'un jpeg
	if len(cleaned) < 2 || cleaned[len(cleaned)-2] != 0xFF || cleaned[len(cleaned)-1] != 0xD9 {
		cleaned = append(cleaned, 0xFF, 0xD9)
	}

	return cleaned, metadataBytes
}

// verif si marker a pas de donne (cas special)
func isStandaloneMarker(marker byte) bool {
	switch marker {
	case 0xD8: // SOI
		return true
	case 0xD9: // EOI
		return true
	case 0xD0, 0xD1, 0xD2, 0xD3, 0xD4, 0xD5, 0xD6, 0xD7: // RSTn c'est genre restart mais je sais pas c'est quoi
		return true
	case 0x01: // TEM temporaire je croi (je sais pas c;est quuoi mais ca existe)
		return true
	}
	return false
}

func shouldKeepSegment(marker byte) bool {
	switch marker {
	case 0xDB: // DQT definition de la table de qunatisation
		return true
	case 0xC0, 0xC1, 0xC2, 0xC3: // SOF debut de frame c'est les plus courant
		return true
	case 0xC4: // DHT definition de la table huffman (je sais pas c'est quoi mais a garder)
		return true
	case 0xDA: // SOS debut du scan
		return true
	case 0xDD: // DRI defini l'interval pour restart
		return true
	case 0xDC: // DNL donne le nombre de ligne
		return true

	// segment a jeter
	case 0xE0, 0xE1, 0xE2, 0xE3, 0xE4, 0xE5, 0xE6, 0xE7: // APP0-APP7
		return false // JFIF, EXIF, XMP, etc.
	case 0xE8, 0xE9, 0xEA, 0xEB, 0xEC, 0xED, 0xEE, 0xEF: // APP8-APP15
		return false // IPTC, Adobe, etc.
	case 0xFE: // COM - Comment
		return false

	// segment chelou qu'il faut garder par secu
	case 0xC5, 0xC6, 0xC7, 0xC9, 0xCA, 0xCB, 0xCD, 0xCE, 0xCF:
		return true

	// on garde si on sait pas c;est quoi pour pa tout peter
	default:
		return true
	}
}
func jpg(pathOfFile string) (map[string]string, error) {

	jpeg := newJpeg()
	content, err := openAndExtractContent(pathOfFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	if !verify_jpg(content, jpeg) {
		return nil, fmt.Errorf("not a valid JPEG file")
	}
	i := 0
	if !handle_Exif(content, &i, jpeg) {
		// log.Fatal("No exif in this file")
	}
	// dedans je parse direct les ifd si il y en a dans l'exif
	if !handle_tiff_header(content, &i, jpeg) {
		// log.Fatal("No tiff in this file")
	}

	for k, v := range jpeg.Tags {
		PrintMetadata(k, v)
	}
	// Si champ gps trouver je format les value et je donne un petit lien openstreemap
	printGPSInfo(jpeg.Tags)

	// En gros je scane si il y des segement hor exif (XMP, IPTC, Comment)
	scan_jpeg_segments(content, jpeg)

	return jpeg.Tags, nil
}

func handle_tiff_header(content []byte, i *int, jpeg *Jpeg) bool {
	if content[*i] == 0x49 && content[*i+1] == 0x49 || content[*i] == 0x4D && content[*i+1] == 0x4D {
		if content[*i] == 0x49 && content[*i+1] == 0x49 {
			jpeg.isLittleEndian = true
			*i += 2
			// fmt.Printf("%x\n", content[*i:*i+2])
		}
		if content[*i] == 0x4D && content[*i+1] == 0x4D {
			jpeg.isLittleEndian = false
			*i += 2
			// fmt.Printf("%x\n", content[*i:*i+2])
		}
	} else {
		// fmt.Println("No tiff header")
		return false
	}

	magic := readUint16(content[*i:*i+2], jpeg.isLittleEndian)
	if magic != 0x002A {
		// fmt.Println("Invalid TIFF magic")
		return false
	}
	// fmt.Printf("Endian: %t, Magic numver: 0x%04X OK\n", jpeg.isLittleEndian, magic)
	*i += 2
	if *i+4 > len(content) {
		return false
	}
	ifdOffset := readUint32(content[*i:*i+4], jpeg.isLittleEndian)
	*i += 4

	// base TIFF = jpeg.exifStartPosition + 4 (marker+size) + len("Exif\0\0")
	tiffBase := jpeg.exifStartPosition + 4 + len("Exif\x00\x00")
	ifdPos := tiffBase + int(ifdOffset)
	if !parseIFD(content, ifdPos, jpeg) {
		// fmt.Println("Unable to parse IFD0")
		return false
	}
	return true
}

func readIFDValue(content []byte, typ uint16, count uint32, valOrOffset []byte, jpeg *Jpeg) string {
	typeSize := map[uint16]uint32{
		1:  1, // BYTE
		2:  1, // ASCII
		3:  2, // SHORT
		4:  4, // LONG
		5:  8, // RATIONAL
		6:  1, // SBYTE (signed byte)
		7:  1, // UNDEFINED
		8:  2, // SSHORT (signed short)
		9:  4, // SLONG (signed long)
		10: 8, // SRATIONAL (signed rational)
		12: 4, // IFD (pointeur vers autre IFD)
	}

	size, ok := typeSize[typ]
	if !ok {
		return ""
	}
	totalSize := size * count

	// Base a partir de laquelle je doi appliquer les offset pour touver les value
	tiffBase := jpeg.exifStartPosition + 4 + len("Exif\x00\x00")

	var data []byte
	if totalSize <= 4 {
		// valeur directement dans les 4 octets si la taille total est inferieur ou = a 4
		data = valOrOffset[:totalSize]
	} else {
		// sinon  recuperer le offset
		offset := int(readUint32(valOrOffset, jpeg.isLittleEndian))
		// si ca depasse la taille du fichier je return
		if tiffBase+offset+int(totalSize) > len(content) {
			return ""
		}
		// si c'est ok
		data = content[tiffBase+offset : tiffBase+offset+int(totalSize)]
	}

	switch typ {
	case 1: // BYTE
		if count == 1 {
			return fmt.Sprintf("%d", data[0])
		}
		var values []string
		for i := uint32(0); i < count && int(i) < len(data); i++ {
			values = append(values, fmt.Sprintf("%d", data[i]))
		}
		return strings.Join(values, ", ")

	case 2: // ASCII
		n := len(data)
		// je prend jusqu'au char \o
		if n > 0 && data[n-1] == 0 {
			data = data[:n-1]
		}
		return string(data)

	case 3: // SHORT (unsigned)
		if count == 1 {
			if len(data) < 2 {
				return ""
			}
			v := readUint16(data[:2], jpeg.isLittleEndian)
			return fmt.Sprintf("%d", v)
		}
		var values []string
		for i := uint32(0); i < count; i++ {
			pos := int(i * 2)
			if pos+2 > len(data) {
				break
			}
			v := readUint16(data[pos:pos+2], jpeg.isLittleEndian)
			values = append(values, fmt.Sprintf("%d", v))
		}
		return strings.Join(values, ", ")

	case 4: // LONG (unsigned)
		if count == 1 {
			if len(data) < 4 {
				return ""
			}
			v := readUint32(data[:4], jpeg.isLittleEndian)
			return fmt.Sprintf("%d", v)
		}
		var values []string
		for i := uint32(0); i < count; i++ {
			pos := int(i * 4)
			if pos+4 > len(data) {
				break
			}
			v := readUint32(data[pos:pos+4], jpeg.isLittleEndian)
			values = append(values, fmt.Sprintf("%d", v))
		}
		return strings.Join(values, ", ")

	case 5: // RATIONAL (unsigned)
		var values []string
		for i := uint32(0); i < count; i++ {
			pos := int(i * 8)
			if pos+8 > len(data) {
				break
			}
			num := readUint32(data[pos:pos+4], jpeg.isLittleEndian)
			den := readUint32(data[pos+4:pos+8], jpeg.isLittleEndian)
			if den == 0 {
				values = append(values, "0/0")
			} else {
				values = append(values, fmt.Sprintf("%d/%d", num, den))
			}
		}
		return strings.Join(values, ", ")

	case 6: // SBYTE (signed byte)
		if count == 1 {
			return fmt.Sprintf("%d", int8(data[0]))
		}
		var values []string
		for i := uint32(0); i < count && int(i) < len(data); i++ {
			values = append(values, fmt.Sprintf("%d", int8(data[i])))
		}
		return strings.Join(values, ", ")

	case 7: // UNDEFINED
		// Certain tag exif indefini on des valeur bizarre donc je le envoi en brut
		return fmt.Sprintf("data:%d bytes", len(data))

	case 8: // SSHORT (signed short)
		if count == 1 {
			if len(data) < 2 {
				return ""
			}
			uv := readUint16(data[:2], jpeg.isLittleEndian)
			sv := int16(uv) // cast en signed
			return fmt.Sprintf("%d", sv)
		}
		var values []string
		for i := uint32(0); i < count; i++ {
			pos := int(i * 2)
			if pos+2 > len(data) {
				break
			}
			uv := readUint16(data[pos:pos+2], jpeg.isLittleEndian)
			sv := int16(uv)
			values = append(values, fmt.Sprintf("%d", sv))
		}
		return strings.Join(values, ", ")

	case 9: // SLONG (signed long)
		if count == 1 {
			if len(data) < 4 {
				return ""
			}
			uv := readUint32(data[:4], jpeg.isLittleEndian)
			sv := int32(uv) // cast en signed
			return fmt.Sprintf("%d", sv)
		}
		var values []string
		for i := uint32(0); i < count; i++ {
			pos := int(i * 4)
			if pos+4 > len(data) {
				break
			}
			uv := readUint32(data[pos:pos+4], jpeg.isLittleEndian)
			sv := int32(uv)
			values = append(values, fmt.Sprintf("%d", sv))
		}
		return strings.Join(values, ", ")

	case 10: // SRATIONAL (signed rational)
		var values []string
		for i := uint32(0); i < count; i++ {
			pos := int(i * 8)
			if pos+8 > len(data) {
				break
			}
			unum := readUint32(data[pos:pos+4], jpeg.isLittleEndian)
			uden := readUint32(data[pos+4:pos+8], jpeg.isLittleEndian)
			snum := int32(unum)
			sden := int32(uden)
			if sden == 0 {
				values = append(values, "0/0")
			} else {
				values = append(values, fmt.Sprintf("%d/%d", snum, sden))
			}
		}
		return strings.Join(values, ", ")

	case 12: // IFD
		if len(data) < 4 {
			return ""
		}
		v := readUint32(data[:4], jpeg.isLittleEndian)
		return fmt.Sprintf("%d", v)

	default:
		return ""
	}
}

func parseIFD(content []byte, pos int, jpeg *Jpeg) bool {
	if pos+2 > len(content) {
		return false
	}
	numEntries := readUint16(content[pos:pos+2], jpeg.isLittleEndian)
	pos += 2

	for e := 0; e < int(numEntries); e++ {
		entryPos := pos + e*12
		if entryPos+12 > len(content) {
			return false
		}

		tag := readUint16(content[entryPos:entryPos+2], jpeg.isLittleEndian)
		typ := readUint16(content[entryPos+2:entryPos+4], jpeg.isLittleEndian)
		count := readUint32(content[entryPos+4:entryPos+8], jpeg.isLittleEndian)
		valOrOffset := content[entryPos+8 : entryPos+12]

		tagName, ok := jpeg.marker[tag]
		if !ok {
			// tagName = fmt.Sprintf("Tag0x%04X", tag)
			// fmt.Println(tagName)
			continue
		}
		// fmt.Println(tagName)
		if tag == 0x8769 || tag == 0x8825 || tag == 0xA005 {
			offset := int(readUint32(valOrOffset, jpeg.isLittleEndian))
			tiffBase := jpeg.exifStartPosition + 4 + len("Exif\x00\x00")
			ifdPos := tiffBase + offset
			// fmt.Printf("Pointeur %s vers offset %d\n", tagName, ifdPos)
			parseIFD(content, ifdPos, jpeg)
			continue
		}
		value := readIFDValue(content, typ, count, valOrOffset, jpeg)

		// poour les tag undifined
		if typ == 7 && value != "" {
			value = decodeUndefinedTag(tag, content, typ, count, valOrOffset, jpeg)
		}

		// if value != "" || typ == 1 || typ == 7 { // Pour voir ce que mon readIFDVALue ne sait pas lire (cas tres specifique comme ca je sais quel champ existe
		// 	jpeg.Tags[tagName] = value
		// }
		if value != "" {
			jpeg.Tags[tagName] = value
		}
	}

	return true
}

func decodeUndefinedTag(tag uint16, content []byte, typ uint16, count uint32, valOrOffset []byte, jpeg *Jpeg) string {
	typeSize := uint32(1) // UNDEFINED = 1 byte per unit
	totalSize := typeSize * count
	tiffBase := jpeg.exifStartPosition + 4 + len("Exif\x00\x00")

	var data []byte
	if totalSize <= 4 {
		data = valOrOffset[:totalSize]
	} else {
		offset := int(readUint32(valOrOffset, jpeg.isLittleEndian))
		if tiffBase+offset+int(totalSize) > len(content) {
			return fmt.Sprintf("data:%d bytes", totalSize)
		}
		data = content[tiffBase+offset : tiffBase+offset+int(totalSize)]
	}

	switch tag {
	case 0x9000: // ExifVersion
		if len(data) == 4 {
			return string(data)
		}
	case 0xa000: // FlashpixVersion
		if len(data) == 4 {
			return string(data)
		}
	case 0x9101: // ComponentsConfiguration
		if len(data) == 4 {
			components := []string{}
			compMap := map[byte]string{0: "-", 1: "Y", 2: "Cb", 3: "Cr", 4: "R", 5: "G", 6: "B"}
			for _, b := range data {
				if name, ok := compMap[b]; ok {
					components = append(components, name)
				}
			}
			return strings.Join(components, ", ")
		}
	case 0x9286: // UserComment
		// Format: [charset(8 bytes)][texte]
		if len(data) >= 8 {
			charset := string(data[:8])
			charset = strings.TrimRight(charset, "\x00")

			textData := data[8:]
			// ASCII ou UNICODE
			if strings.Contains(charset, "ASCII") || charset == "" {
				text := string(textData)
				text = strings.TrimRight(text, "\x00")
				if text != "" {
					return text
				}
			}
		}
		// si pas de texte a decoder,je retourne la taille
		return fmt.Sprintf("data:%d bytes", len(data))
	}

	//  retourner la taille par default
	return fmt.Sprintf("data:%d bytes", len(data))
}

func handle_Exif(content []byte, i *int, jpeg *Jpeg) bool {
	if !go_to_start_exif(content, i, jpeg) {
		// fmt.Println("No exif in this file")
		return false
	}
	if !skip_exif_header_go_to_start_data(content, i) {
		// fmt.Println("No ascii char exif")
		return false
	}
	return true
}

func go_to_start_exif(content []byte, i *int, jpeg *Jpeg) bool {
	for *i+1 < len(content) {
		if content[*i] == 0xFF && content[*i+1] == jpeg.exifStartMarker {
			jpeg.exifStartPosition = *i
			if *i+3 < len(content) {
				jpeg.exifSize = int(content[*i+2])<<8 | int(content[*i+3])
				// je passe le marker et la size
				*i += 4
				return true
			}
			return false
		}
		*i++
	}
	return false
}

func skip_exif_header_go_to_start_data(content []byte, i *int) bool {
	ascii_char_exif := "Exif\000\000"
	j := 0
	for j < len(ascii_char_exif) && *i+j < len(content) {
		if content[*i+j] != ascii_char_exif[j] {
			return false
		}
		j++
	}
	if j == len(ascii_char_exif) {
		*i += j
		return true
	}
	return false
}

func verify_jpg(content []byte, jpeg *Jpeg) bool {
	for i := range content {
		if content[i] == 0xFF && content[i+1] == 0xD8 {
			// fmt.Printf("%x%x\n", content[i], content[i+1])
			jpeg.startMarkerExist = true
			jpeg.SOI = i
			return true
		}
	}
	return false
}

func readUint16(b []byte, little bool) uint16 {
	if little {
		return uint16(b[0]) | uint16(b[1])<<8
	}
	return uint16(b[1]) | uint16(b[0])<<8
}

func readUint32(b []byte, little bool) uint32 {
	if little {
		return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	}
	return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
}

func scan_jpeg_segments(content []byte, jpeg *Jpeg) {
	reader := bytes.NewReader(content)
	soi := make([]byte, 2)
	reader.Read(soi)
	if soi[0] != 0xFF || soi[1] != 0xD8 {
		return
	}

	for {
		markerBuf := make([]byte, 2)
		_, err := reader.Read(markerBuf)
		if err != nil {
			break
		}
		for markerBuf[0] == 0xFF && markerBuf[1] == 0xFF {
			b, _ := reader.ReadByte()
			markerBuf[1] = b
		}
		marker := markerBuf[1]
		if marker == 0xDA {
			// Start of Scan (SOS) in a JPEG file is marked by the hexadecimal code 0xFFDA and signals the beginning of the compressed image data.
			// en gros c;est le debut de l'image donc je break
			break
		}
		sizeBuf := make([]byte, 2)
		if _, err := reader.Read(sizeBuf); err != nil {
			break
		}
		length := int(sizeBuf[0])<<8 | int(sizeBuf[1])
		if length < 2 {
			break
		}
		segmentData := make([]byte, length-2)
		_, err = reader.Read(segmentData)
		if err != nil {
			break
		}
		if marker == 0xE1 {
			xmpHeader := "http://ns.adobe.com/xap/1.0/\x00"
			if len(segmentData) > len(xmpHeader) && string(segmentData[:len(xmpHeader)]) == xmpHeader {
				xmlContent := string(segmentData[len(xmpHeader):])
				parseXMP_JPEG(xmlContent, jpeg)
			}
		}
		if marker == 0xED {
			psHeader := "Photoshop 3.0\x00"
			if len(segmentData) > len(psHeader) && string(segmentData[:len(psHeader)]) == psHeader {
				parseIPTC(segmentData[len(psHeader):], jpeg)
			}
		}
		if marker == 0xFE {
			jpeg.Tags["JPEG Comment"] = string(segmentData)
		}
	}
}

func parseXMP_JPEG(xmp string, jpeg *Jpeg) {
	reTitle := regexp.MustCompile(`dc:title[^>]*>([^<]+)<`)
	reCreator := regexp.MustCompile(`dc:creator[^>]*>([^<]+)<`)
	reDescription := regexp.MustCompile(`dc:description[^>]*>([^<]+)<`)
	reRating := regexp.MustCompile(`xmp:Rating>([^<]+)<`)
	if match := reTitle.FindStringSubmatch(xmp); len(match) > 1 {
		jpeg.Tags["XMP:Title"] = match[1]
	}
	if match := reCreator.FindStringSubmatch(xmp); len(match) > 1 {
		jpeg.Tags["XMP:Creator"] = match[1]
	}
	if match := reDescription.FindStringSubmatch(xmp); len(match) > 1 {
		jpeg.Tags["XMP:Description"] = match[1]
	}
	if match := reRating.FindStringSubmatch(xmp); len(match) > 1 {
		jpeg.Tags["XMP:Rating"] = match[1]
	}
}

func parseIPTC(data []byte, jpeg *Jpeg) {
	for i := 0; i < len(data)-5; i++ {
		if data[i] == 0x1C {
			dataset := data[i+1]
			record := data[i+2]
			size := int(data[i+3])<<8 | int(data[i+4])
			if i+5+size > len(data) {
				break
			}
			value := string(data[i+5 : i+5+size])
			if dataset == 2 {
				switch record {
				case 0x05:
					jpeg.Tags["IPTC:Title"] = value
				case 0x50:
					jpeg.Tags["IPTC:Byline"] = value
				case 0x78:
					jpeg.Tags["IPTC:Caption"] = value
				case 0x19:
					jpeg.Tags["IPTC:Keywords"] = value
				case 0x74:
					jpeg.Tags["IPTC:Copyright"] = value
				}
			}
			i += 4 + size
		}
	}
}
