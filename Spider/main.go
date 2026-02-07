package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
)

type Spider struct {
	rFlag       bool
	lFlag       int
	pFlag       string
	baseUrl     *url.URL
	valid_ext   []string
	visited_url map[string]bool
	banner      string
}

func printHelp() {
	fmt.Println(`
Spider - Minimal Scrapper of images

USAGE:
  spider [-rlp] URL

OPTIONS:
  -r        recursively downloads the images in a URL received as a parameter
  -l        indicates the maximum depth level of the recursive download.(default 5)
  -p        indicates the path where the downloaded files will be saved.(default ./data/ will be used).
  -h        show the help

FORMATS SUPPORTED:
  JPEG/JPG  
  PNG       
  BMP       
  GIF       

EXEMPLES:
  spider  -r http://httpbin.org/links/10/0   # Scrapp Recursively with depth of 5 by default the images on the site
  spider  -r -l 4 [URL]                      # Scrapp Recursively with depth of 4 the images on the site
  spider  -r -l 3 -p ./test/ [URL]           # Recursively retrieves images from the site with depth of 3 and puts them in the ./test folder`)
}

func main() {

	var spider Spider

	helpFlag := flag.Bool("h", false, "show help")
	rFlag := flag.Bool("r", false, "recursively downloads the images in a URL received as a parameter")
	lFlag := flag.Int("l", 5, "indicates the maximum depth level of the recursive download.If not indicated, it will be 5")
	pFlag := flag.String("p", "./data/", "indicates the path where the downloaded files will be saved.If not specified, ./data/ will be used.")

	flag.Parse()
	if *helpFlag {
		printHelp()
		return
	}

	if flag.NArg() != 1 {
		fmt.Println("Url is required")
		os.Exit(1)
	}

	spider.banner = ` █████                          █████       ███                               ████                                                                                
░░███                          ░░███       ░░░                               ░░███                                                                                
 ░███         ██████    ██████  ░███████   ████   ███████ ████████    ██████  ░███      █████   ██████  ████████   ██████   ████████  ████████   ██████  ████████ 
 ░███        ░░░░░███  ███░░███ ░███░░███ ░░███  ███░░███░░███░░███  ███░░███ ░███     ███░░   ███░░███░░███░░███ ░░░░░███ ░░███░░███░░███░░███ ███░░███░░███░░███
 ░███         ███████ ░███ ░░░  ░███ ░███  ░███ ░███ ░███ ░███ ░███ ░███ ░███ ░███    ░░█████ ░███ ░░░  ░███ ░░░   ███████  ░███ ░███ ░███ ░███░███████  ░███ ░░░ 
 ░███      █ ███░░███ ░███  ███ ░███ ░███  ░███ ░███ ░███ ░███ ░███ ░███ ░███ ░███     ░░░░███░███  ███ ░███      ███░░███  ░███ ░███ ░███ ░███░███░░░   ░███     
 ███████████░░████████░░██████  ████ █████ █████░░███████ ████ █████░░██████  █████    ██████ ░░██████  █████    ░░████████ ░███████  ░███████ ░░██████  █████    
░░░░░░░░░░░  ░░░░░░░░  ░░░░░░  ░░░░ ░░░░░ ░░░░░  ░░░░░███░░░░ ░░░░░  ░░░░░░  ░░░░░    ░░░░░░   ░░░░░░  ░░░░░      ░░░░░░░░  ░███░░░   ░███░░░   ░░░░░░  ░░░░░     
                                                 ███ ░███                                                                   ░███      ░███                        
                                                ░░██████                                                                    █████     █████                       
                                                 ░░░░░░                                                                    ░░░░░     ░░░░░                        `

	url := flag.Args()[0]

	spider.rFlag = *rFlag
	spider.lFlag = *lFlag
	spider.pFlag = *pFlag
	spider.valid_ext = []string{".jpg", ".jpeg", ".bmp", ".svg", ".gif", ".png"}

	err := os.MkdirAll(*pFlag, 0755)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	baseUrl, err := extractBaseUrl(url)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	spider.baseUrl = baseUrl
	spider.visited_url = make(map[string]bool)
	spider.visited_url[url] = true

	fmt.Println(spider.banner)
	explore_body(&spider, url, 1)
}
