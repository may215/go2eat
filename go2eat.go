package go2eat

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	mutex = new(sync.Mutex)
)

func init() {}

// Hook function to allow modifications on the link / category before making the request
type before_eat func(data string) (changed_data string)

// Hook function to allow modifications on the data that returned from the request
type after_eat func(data string) (changed_data string)

type Configuration struct {
	Urls               []Url             // List of urls to fetch data for (Can provide this or provide path to json file)
	Method             string            // GET/POST request
	Headers            map[string]string // Set of header for each request
	Timeout            int64             // Each url request time out in milisecond
	Period             int               // All execution time frame
	BeforeEat          before_eat        // Hook function to allow modifications on the link / category before making the request
	AfterEat           after_eat         // Hook function to allow modifications on the data that returned from the request
	UseOsExitSignal    bool              // Allow this library to handle os signal (default is false)
	InsecureSkipVerify bool              // Use in secure connection (default is false)
	FilePath           string            // Full path to json file that contain the list of urls to fetch
	MaxProcess         int               // Max processes that the eater can use
}

// Custom error contruction
type handlerError struct {
	Error   error
	Message string
	Code    int
}

type Url struct {
	Category string
	Link     string
}

func readUrlsList(configuration *Configuration) ([]Url, *handlerError) {
	var urls []Url
	lines, err := readLines(configuration)

	if err.Error != nil {
		return urls, &handlerError{err.Error, "Error reading file", 100}
	}

	var conf = strings.Join(lines, " ")
	b := []byte(conf)
	err.Error = json.Unmarshal(b, &urls)

	if err.Error != nil {
		return urls, &handlerError{err.Error, "Unable to marshal json data", 101}
	}
	return urls, nil
}

// Read json file that contain array of category -> link.
func readLines(configuration *Configuration) ([]string, *handlerError) {
	file, err := os.Open(configuration.FilePath)
	if err != nil {
		return nil, &handlerError{err, "Unable to open file for read", 102}
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, &handlerError{err, "success reading file lines", 0}
}

// Set the main configuration based on the data the user provide or default.
func VerifyConfiguration(configuration *Configuration) (*Configuration, *handlerError) {
	if configuration.Urls == nil {
		if configuration.FilePath != "" {
			config_urls, err := readUrlsList(configuration)
			if err != nil {
				return configuration, &handlerError{err.Error, "Error reading file - " + err.Message, 108}
			}
			var urls = make([]Url, 0)
			for _, url := range config_urls {
				urls = append(urls, url)
			}
			configuration.Urls = urls
		} else {
			return nil, &handlerError{nil, "You must provide list of urls to eat", 103}
		}
	}

	if configuration.Timeout == 0 {
		return nil, &handlerError{nil, "You must provide the process timeout", 104}
	}

	if configuration.UseOsExitSignal {
		if configuration.UseOsExitSignal && configuration.Timeout != 0 && configuration.Period != 0 {
			timeout := make(chan bool, 1)
			go func() {
				<-time.After(time.Duration(configuration.Period) * time.Second)
				timeout <- true
			}()

			go func() {
				<-timeout
				syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
			}()
		}
	}

	return configuration, nil
}

func redirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) >= 3 {
		return errors.New("stopped after 3 redirects")
	}
	return nil
}

// Custom timout dialer, and the main reason is for avoiding closing the response before getting its body
func timeoutDialler(configuration *Configuration) *http.Client {
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: configuration.InsecureSkipVerify},
		Dial: func(netw, addr string) (net.Conn, error) {
			deadline := time.Now().Add(time.Duration(configuration.Period) * time.Millisecond)
			c, err := net.DialTimeout(netw, addr, time.Second)
			if err != nil {
				return nil, err
			}
			c.SetDeadline(deadline)
			return c, nil
		}}
	httpclient := &http.Client{Transport: transport, CheckRedirect: redirectPolicy}
	return httpclient
}

func fetcher(configuration *Configuration, feeds map[string][]string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recover in action: ", r)
			os.Exit(1)
		}
	}()
	var wg sync.WaitGroup

	for _, url := range configuration.Urls {
		wg.Add(1)
		go func(url Url) {
			defer wg.Done()
			client := timeoutDialler(configuration)

			if configuration.BeforeEat != nil {
				url.Link = configuration.BeforeEat(url.Link)
			}

			req, err := http.NewRequest(configuration.Method, url.Link, nil)

			if configuration.Headers != nil {
				for k, v := range configuration.Headers {
					req.Header.Add(k, v)
				}
			}
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			// Check that the server actually sent compressed data
			data, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return
			}
			str := string(data)
			if configuration.AfterEat != nil {
				str = configuration.AfterEat(str)
			}
			mutex.Lock()
			feeds[url.Category] = append(feeds[url.Category], str)
			mutex.Unlock()
		}(url)
	}
	wg.Wait()
}

func returnData(feeds *map[string][]string) *map[string][]string {
	return feeds
}

func EatIt(configuration *Configuration) (map[string][]string, *handlerError) {
	var feeds = make(map[string][]string)
	// Allowing the library to terminate the excution in a timeout condition.
	if configuration.UseOsExitSignal {
		signalChannel := make(chan os.Signal, 2)
		signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
		go func() {
			_ = <-signalChannel
			returnData(&feeds)
			os.Exit(0)
		}()
	}

	if configuration != nil {
		var e *handlerError
		configuration, e = VerifyConfiguration(configuration)
		if e != nil {
			return nil, &handlerError{e.Error, "Unable to verify configuration - " + e.Message, 107}
		}
	} else {
		return nil, &handlerError{nil, "You must provide valid configuration", 107}
	}

	runtime.GOMAXPROCS(configuration.MaxProcess)

	fetcher(configuration, feeds)
	return feeds, nil
}
