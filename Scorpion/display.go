package main

import (
	"fmt"
	"os"
	"strings"
)

var IsTUIMode = false

const (
	Reset   = "\033[0m"
	Cyan    = "\033[36m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Red     = "\033[31m"
	Magenta = "\033[35m"
	Gray    = "\033[90m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
)

const (
	TopLeft     = "┌"
	TopRight    = "┐"
	BottomLeft  = "└"
	BottomRight = "┘"
	Horizontal  = "─"
	Vertical    = "│"
	LeftT       = "├"
	RightT      = "┤"
	Cross       = "┼"
)

func PrintBanner() {
	fmt.Println(Cyan + Bold)
	fmt.Println("╔═══════════════════════════════════════════╗")
	fmt.Println("║        SCORPION METADATA SCANNER          ║")
	fmt.Println("╚═══════════════════════════════════════════╝")
	fmt.Println(Reset)
}

func PrintSeparator() {
	if IsTUIMode {
		fmt.Println(Gray + strings.Repeat("─", 80) + Reset)
	} else {
		fmt.Println("---------------------")
	}
}

func PrintHeader(text string) {
	fmt.Printf("%s%s>> %s%s\n", Cyan, Bold, text, Reset)
}

func PrintFileStart(index, total int, filename string) {
	if IsTUIMode {
		fmt.Printf("\n%s┌─ [%d/%d] %s%s\n", Cyan, index, total, filename, Reset)
		fmt.Printf("%s│%s\n", Cyan, Reset)
	}
}

func PrintFilePath(path string) {
	if IsTUIMode {
		fmt.Printf("%s│ %sPATH:%s %s%s\n", Cyan, Yellow, Reset, Dim, path)
		fmt.Print(Reset)
	} else {
		fmt.Println(path)
	}
}

func PrintMetadataStart() {
	if IsTUIMode {
		fmt.Printf("%s│ %sMETADATA:%s\n", Cyan, Yellow, Reset)
	}
}

func PrintMetadata(key, value string) {
	maxValueLen := 60
	displayValue := value
	if len(value) > maxValueLen {
		displayValue = value[:maxValueLen-3] + "..."
	}

	if IsTUIMode {
		// Format: │ ├─ key: value
		fmt.Printf("%s│ %s├─ %s%-25s%s %s%s%s\n",
			Cyan,    // │
			Gray,    // ├─
			Green,   // key color
			key+":", // key with padding
			Reset,
			Dim, // dim value
			displayValue,
			Reset)
	} else {
		fmt.Printf("[%s]: %s\n", key, value)
	}
}

func PrintImageInfo(info string) {
	if IsTUIMode {
		fmt.Printf("%s│ %s└─ %s%s%s\n",
			Cyan,
			Gray,
			Magenta,
			info,
			Reset)
	} else {
		fmt.Println(info)
	}
}

func PrintFileEnd() {
	if IsTUIMode {
		fmt.Printf("%s└%s%s\n", Cyan, strings.Repeat("─", 79), Reset)
	} else {
		PrintSeparator()
	}
}

func PrintProcessingStart(total int) {
	if IsTUIMode {
		fmt.Printf("\n%s%s>> PROCESSING %d FILES...%s\n\n", Green, Bold, total, Reset)
	}
}

func PrintDone() {
	if IsTUIMode {
		fmt.Printf("\n%s%s>> SCAN COMPLETE%s\n\n", Green, Bold, Reset)
	}
}

func PrintError(message string) {
	if IsTUIMode {
		fmt.Printf("%s[!] ERROR: %s%s\n", Red, message, Reset)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	}
}

func PrintWarning(message string) {
	if IsTUIMode {
		fmt.Printf("%s[!] WARNING: %s%s\n", Yellow, message, Reset)
	} else {
		fmt.Printf("Warning: %s\n", message)
	}
}

func PrintSuccess(message string) {
	if IsTUIMode {
		fmt.Printf("%s[+] %s%s\n", Green, message, Reset)
	} else {
		fmt.Printf("%s\n", message)
	}
}

func PrintCleanResult(fileType, outputPath string, originalSize, cleanedSize, metadataRemoved int) {
	if IsTUIMode {
		if metadataRemoved > 0 {
			fmt.Printf("%s│ %s├─ %sMetadata removed:%s      %s%d bytes%s\n",
				Cyan, Gray, Yellow, Reset, Dim, metadataRemoved, Reset)
		}
		fmt.Printf("%s│ %s├─ %sOriginal size:%s         %s%d bytes%s\n",
			Cyan, Gray, Yellow, Reset, Dim, originalSize, Reset)
		fmt.Printf("%s│ %s├─ %sCleaned size:%s          %s%d bytes%s\n",
			Cyan, Gray, Yellow, Reset, Dim, cleanedSize, Reset)
		fmt.Printf("%s│ %s└─ %s%s cleaned:%s %s%s%s\n",
			Cyan, Gray, Green, fileType, Reset, Dim, outputPath, Reset)
	} else {
		if metadataRemoved > 0 {
			fmt.Printf("Metadata removed: %d bytes\n", metadataRemoved)
		}
		fmt.Printf("%s cleaned: %s (%d bytes → %d bytes)\n",
			fileType, outputPath, originalSize, cleanedSize)
	}
}
