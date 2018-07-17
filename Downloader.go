package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var wg sync.WaitGroup

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter downloadLink: ")
	downloadLink, _ := reader.ReadString('\n')
	downloadLink = strings.TrimSpace(downloadLink)
	_, fileName := filepath.Split(downloadLink)
	name, _ := url.QueryUnescape(fileName)
	start := time.Now()
	res, _ := http.Head(downloadLink)
	maps := res.Header
	length, _ := strconv.Atoi(maps["Content-Length"][0])
	// Get the content length from the header request
	limit := 10                     // 10 Go-routines for the process so each downloads 18.7MB
	lenSub := length / limit        // Bytes for each Go-routine
	diff := length % limit          // Get the remaining for the last request
	body := make([]string, limit+1) // Make up a temporary array to hold the data to be written to the file
	// Started Downloading Parts
	for i := 0; i < limit; i++ {
		wg.Add(1)
		min := lenSub * i       // Min range
		max := lenSub * (i + 1) // Max range
		if i == limit-1 {
			max += diff // Add the remaining bytes in the last request
		}
		go func(min int, max int, i int) {
			client := &http.Client{}
			req, _ := http.NewRequest("GET", downloadLink, nil)
			rangeHeader := "bytes=" + strconv.Itoa(min) + "-" + strconv.Itoa(max-1) // Add the data for the Range header of the form "bytes=0-100"
			req.Header.Add("Range", rangeHeader)
			resp, err := client.Do(req)
			isError(err)
			defer resp.Body.Close()
			reader, _ := ioutil.ReadAll(resp.Body)
			body[i] = string(reader)
			ioutil.WriteFile(strconv.Itoa(i), []byte(string(body[i])), 0x777) // Write to the file i as a byte array
			wg.Done()
		}(min, max, i)
	}
	wg.Wait()
	// Parts Downloaded
	// Started Merge Files
	os.Remove(name)
	f, err := os.OpenFile(strings.TrimSpace(name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	isError(err)
	defer f.Close()
	for i := 0; i < limit; i++ {
		var part = strconv.Itoa(i)
		content, err := ioutil.ReadFile(part) // just pass the file name
		isError(err)
		f.WriteString(string(content))
		os.Remove(part)
	}
	elapsed := time.Since(start)
	fmt.Println("Time Taken: ", elapsed.String())
}

func isError(err error) bool {
	if err != nil {
		fmt.Println(err.Error())
	}
	return (err != nil)
}
