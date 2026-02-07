package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Gif struct {
	width, height uint16
	Tags          map[string]string
}

func newGif() *Gif {
	return &Gif{Tags: make(map[string]string)}
}

func clear_gif(pathOfFile string) {
	content, err := openAndExtractContent(pathOfFile)
	if err != nil {
		log.Fatal(err)
		return
	}

	gif := newGif()
	if !verify_gif(content, gif) {
		log.Fatal("The extension not corresponding with real file format")
		return
	}

	cleanName, cleanedSize := write_without_gif_metadata(pathOfFile, content)
	if cleanName == "" {
		return
	}

	metadataRemoved := len(content) - cleanedSize
	PrintCleanResult("GIF", cleanName, len(content), cleanedSize, metadataRemoved)
}

func isNetscapeExtension(content []byte, i *int) bool {
	if *i >= len(content) {
		return false
	}
	if content[*i] != 11 {
		return false
	}
	if *i+12 > len(content) {
		return false
	}
	return string(content[*i+1:*i+12]) == "NETSCAPE2.0"
}

func write_without_gif_metadata(pathOfFile string, content []byte) (string, int) {
	ext := filepath.Ext(pathOfFile)
	nameOfNewFile := strings.TrimSuffix(pathOfFile, ext)
	nameOfNewFile = nameOfNewFile + "clear" + ext

	f, err := os.Create(nameOfNewFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: can't create file %s: %v\n", nameOfNewFile, err)
		return "", 0
	}
	defer f.Close()

	bytesWritten := 0

	//HEADER
	if len(content) < 13 {
		fmt.Fprintf(os.Stderr, "Error: GIF file too short\n")
		return "", 0
	}
	n, _ := f.Write(content[:13])
	bytesWritten += n

	//GLOBAL COLOR TABLE
	packed := content[10]
	gctSize := 0
	if packed&0x80 != 0 {
		gctBits := (packed & 7) + 1
		gctSize = 3 * (1 << uint(gctBits))
	}
	if 13+gctSize > len(content) {
		fmt.Fprintf(os.Stderr, "Error: GIF corrupted - invalid color table\n")
		return "", 0
	}
	n, _ = f.Write(content[13 : 13+gctSize])
	bytesWritten += n
	i := 13 + gctSize

	for i < len(content) {
		if i >= len(content) {
			break
		}
		b := content[i]
		i++

		switch b {

		case 0x21: // Flag extension
			if i >= len(content) {
				return nameOfNewFile, bytesWritten
			}
			label := content[i]
			i++

			if label == 0xF9 {
				// Graphic Control Extension obliger de garder sinon detruit tout
				if i+4 > len(content) {
					return nameOfNewFile, bytesWritten
				}
				n, _ = f.Write([]byte{0x21, 0xF9})
				bytesWritten += n
				size := content[i]
				i++
				n, _ = f.Write([]byte{size})
				bytesWritten += n
				if i+int(size) > len(content) {
					return nameOfNewFile, bytesWritten
				}
				n, _ = f.Write(content[i : i+int(size)])
				bytesWritten += n
				i += int(size)

				if i >= len(content) {
					return nameOfNewFile, bytesWritten
				}
				n, _ = f.Write([]byte{0x00}) //fin de bloc
				bytesWritten += n
				i++

			} else if label == 0xFF && isNetscapeExtension(content, &i) {
				// Copie en gardant extension netscape pour boucle infini etc sinon casse comporement du gif
				if i >= len(content) {
					return nameOfNewFile, bytesWritten
				}
				n, _ = f.Write([]byte{0x21, 0xFF})
				bytesWritten += n
				size := content[i]
				n, _ = f.Write([]byte{size})
				bytesWritten += n
				i++
				if i+int(size) > len(content) {
					return nameOfNewFile, bytesWritten
				}
				n, _ = f.Write(content[i : i+int(size)])
				bytesWritten += n
				i += int(size)
				for {
					if i >= len(content) {
						return nameOfNewFile, bytesWritten
					}
					subSize := content[i]
					n, _ = f.Write([]byte{subSize})
					bytesWritten += n
					i++
					if subSize == 0 {
						break
					}
					if i+int(subSize) > len(content) {
						return nameOfNewFile, bytesWritten
					}
					n, _ = f.Write(content[i : i+int(subSize)])
					bytesWritten += n
					i += int(subSize)
				}

			} else {
				// sinon autre app on garde pas
				for {
					if i >= len(content) {
						return nameOfNewFile, bytesWritten
					}
					size := content[i]
					i++
					if size == 0 {
						break
					}
					if i+int(size) > len(content) {
						i = len(content)
						return nameOfNewFile, bytesWritten
					}
					i += int(size)
				}
			}

		case 0x2C: // bloc image donc on garde
			start := i - 1
			if i+9 > len(content) {
				return nameOfNewFile, bytesWritten
			}
			i += 9
			// tab;le des couleurs
			if i == 0 || i > len(content) {
				return nameOfNewFile, bytesWritten
			}
			lctFlag := content[i-1] >> 7
			if lctFlag == 1 {
				lctBits := (content[i-1] & 7) + 1
				lctSize := 3 * (1 << uint(lctBits))
				if i+lctSize > len(content) {
					return nameOfNewFile, bytesWritten
				}
				i += lctSize
			}

			if i >= len(content) {
				return nameOfNewFile, bytesWritten
			}
			i++
			// Les données LZW dans un fichier GIF font référence à l’algorithme de compression sans perte LZW (Lempel-Ziv-Welch) utilisé pour réduire la taille du fichier.
			// en gros important
			for {
				if i >= len(content) {
					return nameOfNewFile, bytesWritten
				}
				size := content[i]
				i++
				if size == 0 {
					break
				}
				if i+int(size) > len(content) {
					return nameOfNewFile, bytesWritten
				}
				i += int(size)
			}

			if start >= len(content) || i > len(content) {
				return nameOfNewFile, bytesWritten
			}
			n, _ = f.Write(content[start:i])
			bytesWritten += n

		case 0x3B: // la on indique la fin du fichier
			n, _ = f.Write([]byte{0x3B})
			bytesWritten += n
			return nameOfNewFile, bytesWritten

		default:
			continue
		}
	}

	return nameOfNewFile, bytesWritten
}

func gif(pathOfFile string) (map[string]string, error) {
	gif := newGif()
	content, err := openAndExtractContent(pathOfFile)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	if !verify_gif(content, gif) {
		log.Fatal("Not a valid GIF")
		return nil, err
	}
	parseGifExtensions(content, gif)
	for k, v := range gif.Tags {
		PrintMetadata(k, v)
	}
	PrintImageInfo(fmt.Sprintf("GIF %dx%d", gif.width, gif.height))
	// envoyer peu etre la width et height je sais pas encore
	return gif.Tags, nil
}

func verify_gif(content []byte, gif *Gif) bool {
	if len(content) < 13 ||
		!(string(content[:6]) == "GIF87a" || string(content[:6]) == "GIF89a") {
		return false
	}
	gif.width = uint16(content[6]) | uint16(content[7])<<8
	gif.height = uint16(content[8]) | uint16(content[9])

	packed := content[10]
	gif.Tags["Version"] = string(content[:6])
	gif.Tags["ColorResolution"] = fmt.Sprintf("%d-bit", (packed>>4&7)+1)
	gif.Tags["GlobalColorTable"] = fmt.Sprintf("%d colors", 1<<uint(packed&7))
	gif.Tags["BackgroundColor"] = fmt.Sprintf("%d", content[11])
	gif.Tags["AspectRatio"] = fmt.Sprintf("%d", content[12])
	return true
}

func parseGifComment(gif *Gif, content []byte, i *int) {
	comments := []string{}
	for {
		if *i >= len(content) {
			break
		}
		size := content[*i]
		*i++
		if size == 0 {
			break
		}
		if *i+int(size) <= len(content) {
			comments = append(comments, string(content[*i:*i+int(size)]))
			*i += int(size)
		} else {
			break
		}
	}
	if len(comments) > 0 {
		gif.Tags["GIF_Comment"] = strings.Join(comments, " | ")
		// fmt.Printf("COMMENT: %s\n", gif.Tags["GIF_Comment"])
	}
}

// Parseer les extensions GIF après avoir sauté la Global Color Table
func parseGifExtensions(content []byte, gif *Gif) {
	packedField := content[10]
	gctSize := 0
	if packedField&0x80 != 0 {
		gctBits := (packedField & 7) + 1
		gctSize = 3 * (1 << uint(gctBits))
	}
	i := 13 + gctSize

	frameCount := 0
	gif.Tags["Duration"] = "0ms (0 frames)"
	for i < len(content) {
		blockType := content[i]
		i++

		switch blockType {
		case 0x21: // Extensions
			label := content[i]
			i++
			switch label {
			case 0xFF:
				parseApplicationExtension(gif, content, &i)
			case 0xF9:
				parseGraphicControl(gif, content, &i, &frameCount)
			case 0xFE:
				parseGifComment(gif, content, &i)
			default:
				skipSubBlocks(content, &i)
			}
		case 0x2C: // Image
			skipImageBlock(content, &i)
			frameCount++
		case 0x3B:
			return
		}
	}
	gif.Tags["FrameCount"] = fmt.Sprintf("%d", frameCount)
}

func parseApplicationExtension(gif *Gif, content []byte, i *int) {
	if *i+11 > len(content) {
		skipSubBlocks(content, i)
		return
	}
	size := content[*i]
	*i++
	if size != 11 {
		skipSubBlocks(content, i)
		return
	}

	appId := string(content[*i : *i+11])
	*i += 11
	gif.Tags[fmt.Sprintf("App_%s", appId)] = "présent"
	// fmt.Printf("App: %s\n", appId)
	skipSubBlocks(content, i)
}

func parseGraphicControl(gif *Gif, content []byte, i *int, frameCount *int) {
	if *i+5 > len(content) {
		skipSubBlocks(content, i)
		return
	}
	*i += 2 //size du block + flags
	delay := int(content[*i]) | int(content[*i+1])<<8
	*i += 2
	transIndex := content[*i]
	*i++
	*frameCount++
	if delay > 0 {
		currentTotal := 0
		if dur, exists := gif.Tags["Duration"]; exists {
			fmt.Sscanf(dur, "%d", &currentTotal)
		}
		gif.Tags["Duration"] = fmt.Sprintf("%dms (%d frames)",
			currentTotal+(delay*10), *frameCount)
	}

	if transIndex != 0 && gif.Tags["TransparentColor"] == "" {
		gif.Tags["TransparentColor"] = fmt.Sprintf("%d", transIndex)
	}
	skipSubBlocks(content, i)
}

func skipSubBlocks(content []byte, i *int) {
	for {
		if *i >= len(content) {
			return
		}
		size := content[*i]
		*i++
		if size == 0 {
			return
		}
		*i += int(size)
		if *i > len(content) {
			*i = len(content)
			return
		}
	}
}

func parseNetscape(gif *Gif, content []byte, i *int) {
	// size du block
	if content[*i] != 11 {
		skipSubBlocks(content, i)
		return
	}
	*i++
	if string(content[*i:*i+11]) == "NETSCAPE2.0" {
		*i += 11
		size := content[*i]
		*i++
		if size == 3 {
			gif.Tags["NetscapeLoops"] = fmt.Sprintf("%d",
				int(content[*i+1])|int(content[*i+2])<<8)
			*i += 3
		}
	}
	skipSubBlocks(content, i)
}

func skipImageBlock(content []byte, i *int) {
	if *i+9 > len(content) {
		return
	}
	*i += 9 // image descriptor complet

	// passer la Local Color Table
	lctFlag := content[*i-1] >> 7 // Bit 7 du dernier byte du descriptor
	lctSize := 0
	if lctFlag == 1 {
		lctBits := (content[*i-1] & 7) + 1
		lctSize = 3 * (1 << uint(lctBits))
		*i += lctSize
	}
	// LZW minimum Code Size + sous-blocs
	if *i >= len(content) {
		return
	}
	*i++ // LZW min block
	skipSubBlocks(content, i)
	// fmt.Printf("Image terminée a i=%d\n", *i)
}
