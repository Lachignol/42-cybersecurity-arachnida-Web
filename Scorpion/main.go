package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

func printHelp() {
	fmt.Println(`
Scorpion - Extracteur de métadonnées d'images

USAGE:
  scorpion [options] <fichier1> [fichier2...]
  scorpion -tui

OPTIONS:
  -c        Supprimer les métadonnées (crée <fichier>_clear.<ext>)
  -tui      Mode interactif avec navigation dans les dossiers
  -h        Afficher cette aide

FORMATS SUPPORTÉS:
  JPEG/JPG  - EXIF, IPTC, XMP
  PNG       - tEXt, iTXt, zTXt, tIME, pHYs, iCCP
  BMP       - DIB Header, Color profiles
  GIF       - Comments, Application Extensions

EXEMPLES:
  scorpion image.jpg                 # Affiche les métadonnées
  scorpion -c image.jpg              # Crée image_clear.jpg
  scorpion -tui                      # Mode interactif
  scorpion *.jpg *.png               # Traite plusieurs fichiers`)
}

func main() {

	// checker ce site pour la correc
	// https: //github.com/ianare/exif-samples
	scorpionBanner := `░█▄█░█▀▀░▀█▀░█▀█░█▀▄░█▀█░░░█▀▀░█░█░▀█▀░█▀▄░█▀█░█▀▀░▀█▀░█▀█░█▀▄
░█░█░█▀▀░░█░░█▀█░█░█░█▀█░░░█▀▀░▄▀▄░░█░░█▀▄░█▀█░█░░░░█░░█░█░█▀▄
░▀░▀░▀▀▀░░▀░░▀░▀░▀▀░░▀░▀░░░▀▀▀░▀░▀░░▀░░▀░▀░▀░▀░▀▀▀░░▀░░▀▀▀░▀░▀`
	fmt.Println(scorpionBanner)

	clearFlag := flag.Bool("c", false, "for clear metadata of file")
	tuiFlag := flag.Bool("tui", false, "start interactive TUI mode")
	helpFlag := flag.Bool("h", false, "show help")
	flag.Parse() // Parse les arguments fournis par l'utilisateur

	if *helpFlag {
		printHelp()
		return
	}

	if *tuiFlag {
		IsTUIMode = true
		runTUI()
		return
	}

	if flag.NArg() < 1 {
		fmt.Println("require at least one picture path (Format accepted :[.jpg/jpeg /.bmp /.gif /.png])")
		os.Exit(1)
	}

	var allFilesPath []string
	for _, n := range flag.Args() {
		allFilesPath = append(allFilesPath, n)
	}

	for _, v := range allFilesPath {
		launch_corresponding_func(v, *clearFlag)
	}
}

// struct avec pointeur de fonction on va dire
type ImageHandler struct {
	display func(string) (map[string]string, error) // je met la signature de la fonction que je voudrai attacher
	clear   func(string)                            //idem
}

// je fait une map avec  comme clef une string et comme vlue ma struct d'avant comme ca je rentre juste l'extension et j'aurai les func qui von bien
var imageHandlers = map[string]ImageHandler{
	".jpg":  {jpg, clear_jpg},
	".jpeg": {jpg, clear_jpg},
	".bmp":  {bmp, clear_bmp},
	".gif":  {gif, clear_gif},
	".png":  {png, clear_png},
}

func launch_corresponding_func(pathOfFile string, clearFlag bool) {
	ext := strings.ToLower(path.Ext(pathOfFile))
	handler, ok := imageHandlers[ext]
	if !ok {
		fmt.Println("Format not supported")
		return
	}

	if !IsTUIMode {
		fmt.Println(pathOfFile)
	}

	if clearFlag {
		handler.clear(pathOfFile)
	} else {
		_, err := handler.display(pathOfFile)
		if err != nil {
			PrintError(fmt.Sprintf("%v", err))
		}
	}
}

func runTUI() {
	// reader pour lire les entree user
	reader := bufio.NewReader(os.Stdin)
	// banniere special tui
	PrintBanner()

	// chercher le bon dossier
	folder := interactiveFolderSelect(reader)
	if folder == "" {
		PrintError("No folder selected, exiting")
		return
	}

	// une fois le dossier choisi je recupere toute les images pour cree la list
	files, err := findImageFiles(folder)
	if err != nil {
		PrintError(fmt.Sprintf("Error scanning folder: %v", err))
		return
	}
	if len(files) == 0 {
		PrintWarning("No image files found")
		return
	}

	// je lance le menu pour choisir les fichier que je veu
	selectedFiles := interactiveFileSelect(reader, files)
	if len(selectedFiles) == 0 {
		return
	}

	// je demande si l;user veu clear les metadata
	fmt.Printf("\n%s>> Clear metadata? (y/N):%s ", Yellow+Bold, Reset)
	ans, _ := reader.ReadString('\n')
	clear := strings.TrimSpace(strings.ToLower(ans)) == "y"

	// Si oui je lance une preview pour demander confirmation
	if clear {
		fmt.Printf("\n%s>> METADATA PREVIEW (will be deleted):%s\n", Yellow+Bold, Reset)
		for _, f := range selectedFiles {
			PrintFileStart(1, 1, filepath.Base(f))
			PrintMetadataStart()
			launch_corresponding_func(f, false) // Afficher sans supprimer
			PrintFileEnd()
		}

		// et la confirmation final
		fmt.Printf("\n%s>> Confirm deletion? (y/N):%s ", Red+Bold, Reset)
		confirm, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(confirm)) != "y" {
			PrintWarning("Operation cancelled")
			return
		}
	}

	// je traite les fichiers choisi
	PrintProcessingStart(len(selectedFiles))
	for idx, f := range selectedFiles {
		// Afficher le header
		PrintFileStart(idx+1, len(selectedFiles), filepath.Base(f))
		PrintFilePath(f)
		PrintMetadataStart()
		// je traite la suprresion
		launch_corresponding_func(f, clear) //clear a false si pas clear sinon true et donc je supprime
		PrintFileEnd()
	}
	PrintDone()
}

func interactiveFolderSelect(reader *bufio.Reader) string {
	// homeDir, err := os.Getwd() // demarer du repertoire courant
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir, _ = os.Getwd()
	}
	currentDir := homeDir
	page := 0
	itemsPerPage := 10

	for {
		// print le repertoire actuel
		fmt.Printf("\n%s>> CURRENT DIR:%s %s%s%s\n", Cyan+Bold, Reset, Dim, currentDir, Reset)
		// list de tout les sous rep
		dirs := listSubdirectories(currentDir)

		// print les opts
		fmt.Printf("\n%sOPTIONS:%s\n", Yellow, Reset)
		fmt.Printf("  %s[0]%s  Select this directory\n", Green, Reset)
		if currentDir != "/" {
			fmt.Printf("  %s[..]%s Go to parent directory\n", Green, Reset)
		}

		// afficher les sous rep si present
		if len(dirs) > 0 {
			totalPages := (len(dirs) + itemsPerPage - 1) / itemsPerPage

			if page >= totalPages {
				page = totalPages - 1
			}
			if page < 0 {
				page = 0
			}

			startIdx := page * itemsPerPage
			endIdx := startIdx + itemsPerPage
			if endIdx > len(dirs) {
				endIdx = len(dirs)
			}

			// print les dossier de la page ou on est
			fmt.Printf("\n%sDIRECTORIES%s %s[page %d/%d]:%s\n", Yellow, Reset, Dim, page+1, totalPages, Reset)
			for i := startIdx; i < endIdx; i++ {
				// print dossier son number
				fmt.Printf("  %s[%d]%s %s\n", Green, i+1, Reset, filepath.Base(dirs[i])+"/")
			}

			// afficher opts de nav si plus de 1 page
			if totalPages > 1 {
				fmt.Printf("\n%sNAVIGATION:%s\n", Yellow, Reset)
				if page > 0 {
					fmt.Printf("  %s[p]%s Previous page\n", Green, Reset)
				}
				if page < totalPages-1 {
					fmt.Printf("  %s[n]%s Next page\n", Green, Reset)
				}
			}
		} else {
			fmt.Printf("\n%s(No subdirectories)%s\n", Dim, Reset)
		}

		// lire input de l'user
		fmt.Printf("\n%s>%s ", Cyan+Bold, Reset)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		// [0]: choisir rep courrant
		if input == "0" {
			PrintSuccess(fmt.Sprintf("Selected: %s", currentDir))
			return currentDir
		}

		//[..]: remonter au rep parent
		if input == ".." {
			parent := filepath.Dir(currentDir)
			if parent != currentDir {
				// si rep courant == racine revenir a 0 et pa boucler
				currentDir = parent
				page = 0
			}
			continue
		}

		//[n]: next page
		if input == "n" || input == "N" {
			totalPages := (len(dirs) + itemsPerPage - 1) / itemsPerPage
			if page < totalPages-1 {
				page++
			} else {
				PrintWarning("Already on last page")
			}
			continue
		}

		//[p]: prev page
		if input == "p" || input == "P" {
			if page > 0 {
				page--
			} else {
				PrintWarning("Already on first page")
			}
			continue
		}

		//choisir un number pour aller ver un sous rep
		if num, err := strconv.Atoi(input); err == nil {
			if num > 0 && num <= len(dirs) {
				currentDir = dirs[num-1]
				page = 0
				continue
			} else {
				PrintError(fmt.Sprintf("Invalid number (1-%d)", len(dirs)))
				continue
			}
		}

		// si l'input est chemin absolu aller directement au rep
		if filepath.IsAbs(input) {
			if stat, err := os.Stat(input); err == nil && stat.IsDir() {
				currentDir = input
				page = 0
				continue
			} else {
				PrintError("Invalid path or not a directory")
				continue
			}
		}
		// si op
		PrintError("Invalid choice. Use: 0, .., number, p, n, or absolute path")
	}
}

func listSubdirectories(dirPath string) []string {
	var dirs []string

	// lire le rep
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return dirs
	}

	// affiche que les rep non cache donc pas de .
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			dirs = append(dirs, filepath.Join(dirPath, entry.Name()))
		}
	}
	return dirs
}

func interactiveFileSelect(reader *bufio.Reader, files []string) []string {
	fmt.Printf("\n%s>> FILES FOUND:%s\n", Yellow+Bold, Reset)
	fmt.Printf("%sUse: 'all', '1,3-5,7' or 'q' to quit%s\n\n", Dim, Reset)

	for i, f := range files {
		fmt.Printf("  %s[%d]%s %s\n", Green, i+1, Reset, filepath.Base(f))
	}

	fmt.Printf("\n%s>%s ", Cyan+Bold, Reset)
	input, _ := reader.ReadString('\n')
	choice := strings.TrimSpace(strings.ToLower(input))

	if choice == "q" || choice == "quit" {
		PrintWarning("Cancelled")
		return nil
	}

	if strings.Contains(choice, "all") {
		PrintSuccess(fmt.Sprintf("All %d files selected", len(files)))
		return files
	}

	var selected []int
	// parts := strings.Split(choice, ",")
	parts := strings.SplitSeq(choice, ",")

	for part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// verif si input est une plage si oui si elle est valid
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) == 2 {
				start, _ := strconv.Atoi(rangeParts[0])
				end, _ := strconv.Atoi(rangeParts[1])

				// je rentre un a un les num de la plage donner par l'user
				for i := start; i <= end && i <= len(files); i++ {
					if i > 0 {
						selected = append(selected, i-1)
					}
				}
			}
		} else {
			// sinon num normal
			num, _ := strconv.Atoi(part)
			if num > 0 && num <= len(files) {
				selected = append(selected, num-1)
			}
		}
	}

	// liste des fichier choisi
	var result []string
	for _, idx := range selected {
		if idx < len(files) {
			result = append(result, files[idx])
		}
	}

	// Si pas de fichier valid je traite tous par default
	if len(result) == 0 {
		PrintWarning("No valid selection. Processing all files")
		return files
	}

	PrintSuccess(fmt.Sprintf("%d file(s) selected", len(result)))
	return result
}

func findImageFiles(root string) ([]string, error) {
	var files []string

	// recherche recursive dans le dossier fonction "native on va dire"
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// j;ignore si dossier je veu que les fichier
			return nil
		}
		// jajoute que si l'extension est une ext que je traite
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".jpg", ".jpeg", ".bmp", ".gif", ".png":
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
