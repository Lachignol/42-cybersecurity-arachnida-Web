package main

import (
	"errors"
	"fmt"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

func isValidURL(rawURL string) bool {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func explore_body(spider *Spider, currentUrl string, idx int) {
	body_html, err := fetch_and_extract_body(currentUrl)
	if err != nil {
		return
	}

	fmt.Println("LINK:", currentUrl, "| DEPTH:", idx, strings.Repeat(`▄▖`, idx))

	var links []string
	for n := range body_html.Descendants() {
		download_images(n, spider.baseUrl, spider.pFlag, spider.valid_ext)
		if mustLaunchRecursion(spider, idx) {
			if link := extract_url(n, spider); len(link) > 0 {
				links = append(links, link)
				continue
			}
		}
	}

	if mustLaunchRecursion(spider, idx) {
		launch_recursion(links, spider, idx+1)
	}

}

func mustLaunchRecursion(spider *Spider, idx int) bool {
	return spider.rFlag && idx < spider.lFlag
}

func launch_recursion(links []string, spider *Spider, idx int) {

	for _, n := range links {
		explore_body(spider, n, idx)
	}
}

func extract_url(currentNode *html.Node, spider *Spider) string {
	if currentNode.Type == html.ElementNode && currentNode.DataAtom == atom.A {
		for _, a := range currentNode.Attr {
			if a.Key == "href" {
				// fmt.Println("----------------------One URL----------------------")
				// fmt.Println("Before compose absolute path:")
				absolutePath, err := createAbsolutePathIfIsNot(spider.baseUrl, a.Val)
				if err != nil || !isValidURL(absolutePath) {
					continue
				}
				// fmt.Println("After compose absoulute path:")
				// fmt.Println(absolutePath)
				// fmt.Println("---------------------------------------------------")
				// fmt.Println(currentUrl)
				// if spider.visited_url[absolutePath] {
				// 	break
				// }
				// body_html, err := fetch_and_extract_body(absolutePath)
				// if err != nil {
				// 	continue
				// }
				// fmt.Println(a.Val)

				if !spider.visited_url[absolutePath] {
					// fmt.Println(a.Val)
					spider.visited_url[absolutePath] = true
					return absolutePath
				}
				return ""
			}
		}
	}
	return ""
}

func download_images(currentNode *html.Node, baseUrl *url.URL, downloadDirectory string, validExt []string) {

	if currentNode.Type == html.ElementNode && currentNode.DataAtom == atom.Img {
		for _, a := range currentNode.Attr {
			if a.Key == "src" {
				// fmt.Println("Extracting image:")
				// fmt.Println("Before compose absolute path:", a.Val)
				absolutePath, err := createAbsolutePathIfIsNot(baseUrl, a.Val)
				if err != nil {
					continue
					// log.Fatal(err)
				}
				// fmt.Println("After compose absolute path:", absolutePath)
				// fmt.Println("Extension", filepath.Ext(absolutePath))
				if slices.Contains(validExt, filepath.Ext(absolutePath)) {
					// fmt.Println("Record image:", absolutePath)
					writeImgFile(absolutePath, downloadDirectory)
				}
			}

		}

	}
}

func writeImgFile(absolutePath string, downloadDirectory string) {
	fileName := path.Base(absolutePath)
	filePath := filepath.Join(downloadDirectory, fileName)

	f, err := os.Create(filePath)
	if err != nil {
		log.Printf("Error creating file %s: %v\n", filePath, err)
		return
	}
	defer f.Close()

	resp, err := http.Get(absolutePath)
	if err != nil {
		log.Printf("Error downloading %s: %v\n", absolutePath, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Bad status for %s: %s\n", absolutePath, resp.Status)
		return
	}

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		log.Printf("Error writing file %s: %v\n", filePath, err)
		return
	}
}

func fetch_and_extract_body(url string) (*html.Node, error) {

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer resp.Body.Close()
	body_html, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return body_html, nil

}

func createAbsolutePathIfIsNot(baseUrl *url.URL, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("path is empty")
	}

	rel, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	abs := baseUrl.ResolveReference(rel)
	if baseUrl.Host != abs.Host {
		return "", errors.New("not the same domain")
	}

	return abs.String(), nil
}

func extractBaseUrl(completeUrl string) (*url.URL, error) {
	base, err := url.Parse(completeUrl)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return base, nil
}
