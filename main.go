package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

type urlProcess struct {
	url  string
	wait *sync.WaitGroup
}

type counter struct {
	count int
	lock  *sync.Mutex
}

func newCounter() *counter {
	return &counter{lock: &sync.Mutex{}}
}

func (c *counter) Add(count int) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.count += count
}

func (c *counter) GetCount() int {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.count
}

func main() {
	parallel := flag.Int("k", 5, "max count parallel goroutines")
	findWord := flag.String("q", "Go", "Word")
	fileName := flag.String("f", "", "File name")
	flag.Parse()
	tokens := make(chan struct{}, *parallel)
	f, err := os.Open(*fileName)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error while reading file: %s", err)
	}
	urls := make(chan urlProcess)
	go func() {
		var waitGroup sync.WaitGroup
		for scanner.Scan() {
			waitGroup.Add(1)
			urls <- urlProcess{scanner.Text(), &waitGroup}
		}
		waitGroup.Wait()
		close(urls)
	}()

	counter := newCounter()
	for url := range urls {
		go handleUrl(url, *findWord, tokens, counter)
	}
	close(tokens)
	fmt.Printf("Total: %d\n", counter.GetCount())
}

func getBody(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return string(body), err
}

func handleUrl(process urlProcess, findWord string, tokens chan struct{}, c *counter) {
	tokens <- struct{}{}
	if body, err := getBody(process.url); err == nil {
		count := strings.Count(body, findWord)
		fmt.Printf("Count for %s: %d\n", process.url, count)
		c.Add(count)
	} else {
		fmt.Printf("processing error - %s\n", err)
	}
	<-tokens
	process.wait.Done()
}
