package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

const maxThreads = 100 // 設置最大線程數

type result struct {
	index int
	url   string
	title string
	err   error
}

func getTitle(urlStr string, index int, proxyURL *url.URL) result {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "http://" + urlStr
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return result{index: index, url: urlStr, err: fmt.Errorf("failed to create request: %w", err)}
	}

	// 添加自訂請求頭
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 YaBrowser/23.3.1.895 Yowser/2.5 Safari/537.36")
	req.Header.Set("Accept-Language", "ru,en;q=0.9,en-US;q=0.8")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if proxyURL != nil {
		tr.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return result{index: index, url: urlStr, err: fmt.Errorf("request failed: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return result{index: index, url: urlStr, err: fmt.Errorf("HTTP error: %s", resp.Status)}
	}

	// 檢測字符編碼
	bodyReader, err := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
	if err != nil {
		return result{index: index, url: urlStr, err: fmt.Errorf("failed to create reader: %w", err)}
	}

	// 創建一個 UTF-8 reader
	utfReader := transform.NewReader(bodyReader, unicode.UTF8.NewDecoder())

	// 使用 goquery 解析 HTML
	doc, err := goquery.NewDocumentFromReader(utfReader)
	if err != nil {
		return result{index: index, url: urlStr, err: fmt.Errorf("failed to parse HTML: %w", err)}
	}

	title := doc.Find("title").First().Text()
	if title == "" {
		return result{index: index, url: urlStr, err: fmt.Errorf("no title found")}
	}

	return result{index: index, url: urlStr, title: title}
}

func printHelp() {
	fmt.Println("Usage: TitleFinder.exe -l <file> [-o <output>] [-p <proxy>] [-t <threads>]")
	fmt.Println("\nOptions:")
	fmt.Println("  -l <file>    Path to the input file containing URLs (required)")
	fmt.Println("  -o <output>  Path to the output file (optional). If not provided, output will be printed to the console.")
	fmt.Println("  -p <proxy>   Proxy URL to use for HTTP requests (optional). Format: [http://]host:port")
	fmt.Println("               If protocol is not specified, http:// will be used by default.")
	fmt.Println("  -t <threads> Number of concurrent threads (optional, default 10)")
	fmt.Println("  -h           Display this help message")
}

type job struct {
	index int
	url   string
}

func worker(id int, jobs <-chan job, results chan<- result, proxyURL *url.URL) {
	_ = id
	for j := range jobs {
		results <- getTitle(j.url, j.index, proxyURL)
	}
}

func main() {
	filePath := flag.String("l", "", "Path to the txt file to be loaded")
	outputPath := flag.String("o", "", "Path to the output file (optional)")
	proxy := flag.String("p", "", "Proxy URL to use for HTTP requests (optional)")
	threads := flag.Int("t", 10, "Number of concurrent threads (default 10, max 100)")
	help := flag.Bool("h", false, "Display help message")
	flag.Parse()

	// 只有在沒有提供任何參數或者明確要求幫助時才顯示幫助信息
	if flag.NFlag() == 0 || *help {
		printHelp()
		return
	}

	// 檢查是否提供了必需的 -l 參數
	if *filePath == "" {
		log.Fatal("Please provide the path to the txt file using -l parameter")
	}

	// 檢查並限制線程數
	if *threads > maxThreads {
		log.Printf("[!] Warning: Thread count exceeds maximum allowed (%d). Setting to max.\n", maxThreads)
		*threads = maxThreads
	}

	var proxyURL *url.URL
	var err error
	if *proxy != "" {
		// 如果用戶沒有指定協議，預設使用 http://
		if !strings.HasPrefix(*proxy, "http://") && !strings.HasPrefix(*proxy, "https://") {
			*proxy = "http://" + *proxy
		}
		proxyURL, err = url.Parse(*proxy)
		if err != nil {
			log.Fatalf("Invalid proxy URL: %s", err)
		}
	}

	file, err := os.Open(*filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %s", err)
	}
	defer file.Close()

	urls := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		urls = append(urls, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %s", err)
	}

	results := make(chan result, len(urls))
	jobs := make(chan job, len(urls))

	// 啟動 worker
	for w := 1; w <= *threads; w++ {
		go worker(w, jobs, results, proxyURL)
	}

	// 發送任務
	for i, url := range urls {
		jobs <- job{index: i, url: url}
	}
	close(jobs)

	// 收集結果
	resultMap := make(map[int]result)
	for i := 0; i < len(urls); i++ {
		res := <-results
		resultMap[res.index] = res
	}

	var output *os.File
	var outputFile io.Writer

	if *outputPath != "" {
		output, err = os.Create(*outputPath)
		if err != nil {
			log.Fatalf("Failed to create output file: %s", err)
		}
		defer output.Close()
		outputFile = output
	} else {
		outputFile = os.Stdout
	}

	for i := 0; i < len(resultMap); i++ {
		res := resultMap[i]
		if res.err != nil {
			fmt.Fprintf(outputFile, "[-] %s: %s\n", res.url, res.err)
		} else {
			fmt.Fprintf(outputFile, "[+] %s: %s\n", res.url, res.title)
		}
	}
}
