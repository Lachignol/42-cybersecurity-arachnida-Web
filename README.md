# Arachnida - Spider & Scorpion

[ 🇬🇧 English ](README_EN.md)

Ce dépôt contient deux outils développés en Go pour la manipulation et la récupération d'images : **Spider** et **Scorpion**.

## Table des Matières
- [Spider](#spider)
  - [Description](#description)
  - [Installation](#installation)
  - [Utilisation](#utilisation)
- [Scorpion](#scorpion)
  - [Description](#description-1)
  - [Installation](#installation-1)
  - [Utilisation](#utilisation-1)

---

## Spider




https://github.com/user-attachments/assets/1d0e5b75-461a-469f-94d0-4f20dd524e68



### Description
**Spider** est un web scraper d'images. Il permet de parcourir un site web de manière récursive pour télécharger toutes les images qu'il contient. Il supporte divers formats d'images (JPG, PNG, BMP, GIF, SVG) et permet de contrôler la profondeur de la recherche récursive.

### Installation

Pour compiler le programme, assurez-vous d'avoir [Go](https://go.dev/dl/) installé, puis exécutez les commandes suivantes :

```bash
cd Spider
go build -o spider
```

Cela créera un exécutable nommé `spider` dans le dossier `Spider`.

### Utilisation

La syntaxe générale est la suivante :

```bash
./spider [OPTIONS] <URL>
```

#### Options

| Option | Description | Valeur par défaut |
|--------|-------------|-------------------|
| `-r`   | Active le téléchargement récursif. | Désactivé |
| `-l`   | Définit la profondeur maximale de la récursion. | `5` |
| `-p`   | Spécifie le dossier de destination pour les fichiers téléchargés. | `./data/` |
| `-h`   | Affiche l'aide. | |

#### Exemples

Télécharger récursivement les images d'un site avec une profondeur par défaut (5) :
```bash
./spider -r http://exemple.com
```

Télécharger avec une profondeur de 3 et sauvegarder dans un dossier spécifique :
```bash
./spider -r -l 3 -p mes_images http://exemple.com
```

---

## Scorpion


https://github.com/user-attachments/assets/89146982-f6cb-4353-87cd-47b91dfc8c3e


### Description
**Scorpion** est un outil d'analyse de métadonnées d'images. Il est capable d'extraire et d'afficher les métadonnées (EXIF, IPTC, XMP, etc.) de fichiers images. Il inclut également une fonctionnalité pour supprimer ces métadonnées et un mode interactif (TUI) pour naviguer et sélectionner des fichiers.

Formats supportés : JPEG/JPG, PNG, BMP, GIF.

### Installation

Pour compiler le programme, placez-vous dans le dossier `Scorpion` et lancez la compilation :

```bash
cd Scorpion
go build -o scorpion
```

Cela créera un exécutable nommé `scorpion` dans le dossier `Scorpion`.

### Utilisation

La syntaxe générale est la suivante :

```bash
./scorpion [OPTIONS] <fichier1> [fichier2...]
```

#### Options

| Option | Description |
|--------|-------------|
| `-c`   | Supprime les métadonnées du fichier (crée une copie nommée `_clear`). |
| `-tui` | Lance le mode interactif (interface textuelle) pour naviguer et sélectionner des images. |
| `-h`   | Affiche l'aide. |

#### Exemples

Afficher les métadonnées d'une image :
```bash
./scorpion image.jpg
```

Afficher les métadonnées de plusieurs images :
```bash
./scorpion photo1.png photo2.jpg
```

Supprimer les métadonnées d'une image (crée une copie sans métadonnées) :
```bash
./scorpion -c image.jpg
```

Lancer le mode interactif pour naviguer dans vos dossiers :
```bash
./scorpion -tui
```
