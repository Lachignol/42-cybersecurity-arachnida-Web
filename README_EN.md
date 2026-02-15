# Arachnida - Spider & Scorpion

[ 🇫🇷 Français ](README.md)

This repository contains two tools developed in Go for image manipulation and retrieval: **Spider** and **Scorpion**.

## Table of Contents
- [Spider](#spider)
  - [Description](#description)
  - [Installation](#installation)
  - [Usage](#usage)
- [Scorpion](#scorpion)
  - [Description](#description-1)
  - [Installation](#installation-1)
  - [Usage](#usage-1)

---

## Spider

https://github.com/user-attachments/assets/1d0e5b75-461a-469f-94d0-4f20dd524e68

### Description
**Spider** is an image web scraper. It allows you to recursively crawl a website to download all images it contains. It supports various image formats (JPG, PNG, BMP, GIF, SVG) and allows control over the recursion depth.

### Installation

To compile the program, ensure you have [Go](https://go.dev/dl/) installed, then run the following commands:

```bash
cd Spider
go build -o spider
```

This will create an executable named `spider` in the `Spider` directory.

### Usage

The general syntax is as follows:

```bash
./spider [OPTIONS] <URL>
```

#### Options

| Option | Description | Default Value |
|--------|-------------|-------------------|
| `-r`   | Enables recursive downloading. | Disabled |
| `-l`   | Sets the maximum recursion depth. | `5` |
| `-p`   | Specifies the destination folder for downloaded files. | `./data/` |
| `-h`   | Displays help. | |

#### Examples

Recursively download images from a site with default depth (5):
```bash
./spider -r http://example.com
```

Download with a depth of 3 and save to a specific folder:
```bash
./spider -r -l 3 -p my_images http://example.com
```

---

## Scorpion


https://github.com/user-attachments/assets/89146982-f6cb-4353-87cd-47b91dfc8c3e

### Description
**Scorpion** is an image metadata analysis tool. It extracts and displays metadata (EXIF, IPTC, XMP, etc.) from image files. It also includes a feature to remove this metadata and an interactive mode (TUI) for navigating and selecting files.

Supported formats: JPEG/JPG, PNG, BMP, GIF.

### Installation

To compile the program, navigate to the `Scorpion` directory and run the compilation:

```bash
cd Scorpion
go build -o scorpion
```

This will create an executable named `scorpion` in the `Scorpion` directory.

### Usage

The general syntax is as follows:

```bash
./scorpion [OPTIONS] <file1> [file2...]
```

#### Options

| Option | Description |
|--------|-------------|
| `-c`   | Removes metadata from the file (creates a copy named `_clear`). |
| `-tui` | Launches interactive mode (Text User Interface) to navigate and select images. |
| `-h`   | Displays help. |

#### Examples

Display metadata for an image:
```bash
./scorpion image.jpg
```

Display metadata for multiple images:
```bash
./scorpion photo1.png photo2.jpg
```

Remove metadata from an image (creates a copy without metadata):
```bash
./scorpion -c image.jpg
```

Launch interactive mode to navigate through your folders:
```bash
./scorpion -tui
```
