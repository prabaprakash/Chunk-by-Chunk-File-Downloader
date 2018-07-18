package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type chunks struct {
	min int
	max int
	id  int
	url string
	dir string
}

func worker(i int, jobs <-chan chunks, results chan<- int) {
	for j := range jobs {
		//fmt.Println("worker", i, "started  job", j.id)
		client := &http.Client{}
		req, _ := http.NewRequest("GET", j.url, nil)
		rangeHeader := "bytes=" + strconv.Itoa(j.min) + "-" + strconv.Itoa(j.max-1) // Add the data for the Range header of the form "bytes=0-100"
		req.Header.Add("Range", rangeHeader)
		resp, err := client.Do(req)
		isError(err)
		defer resp.Body.Close()
		reader, _ := ioutil.ReadAll(resp.Body)
		location := strings.Replace(path.Join(j.dir, strconv.Itoa(j.id)), "/", "\\", -1)
		ioutil.WriteFile(location, []byte(string(reader)), 0x777) // Write to the file i as a byte array
		resp.Body.Close()
		//fmt.Println("worker", i, "finished job", j.id)
		results <- j.id
	}
}

func main() {
	jobs := make(chan chunks, 100)
	results := make(chan int, 100)
	dir, err := ioutil.TempDir(os.TempDir(), "prefix")
	//fmt.Println(strings.Replace(path.Join(dir, "2222"), "/", "\\", -1))
	isError(err)
	if len(os.Args) == 1 {
		fmt.Printf("pass the url")
		return
	}
	downloadLink := os.Args[1]
	downloadLink = strings.TrimSpace(downloadLink)
	_, fileName := filepath.Split(downloadLink)
	name, _ := url.QueryUnescape(fileName)
	start := time.Now()
	res, _ := http.Head(downloadLink)
	maps := res.Header
	length, _ := strconv.Atoi(maps["Content-Length"][0])
	// println(length)
	// Get the content length from the header request
	limit := 10              // 10 Go-routines for the process so each downloads 18.7MB
	lenSub := length / limit // Bytes for each Go-routine
	diff := length % limit   // Get the remaining for the last request
	// Started Downloading Parts
	for i := 0; i < 3; i++ {
		go worker(i, jobs, results)
	}

	for i := 0; i < limit; i++ {
		min := lenSub * i       // Min range
		max := lenSub * (i + 1) // Max range
		if i == limit-1 {
			max += diff // Add the remaining bytes in the last request
		}
		jobs <- chunks{id: i, min: min, max: max, url: downloadLink, dir: dir}
	}
	close(jobs)
	// Finally we collect all the results of the work.
	for a := 0; a < limit; a++ {
		<-results
	}
	mergeParts(name, limit, dir)
	fmt.Println("Time Taken: ", time.Since(start).String())
}
func mergeParts(name string, limit int, dir string) {
	os.Remove(name)
	f, err := os.OpenFile(strings.TrimSpace(name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	isError(err)
	defer f.Close()
	for i := 0; i < limit; i++ {
		var part = strconv.Itoa(i)
		location := strings.Replace(path.Join(dir, part), "/", "\\", -1)
		content, err := ioutil.ReadFile(location) // just pass the file name
		isError(err)
		f.WriteString(string(content))
		os.Remove(location)
	}
}
func isError(err error) bool {
	if err != nil {
		fmt.Println(err.Error())
	}
	return (err != nil)
}
