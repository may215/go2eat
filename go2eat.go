package go2eat

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

var (
	urls    []Url
	method  string
	headers map[string]string
	timeout int64
	period  int
)

func init() {
	flag.Int64Var(&timeout, "t", 0, "Each request time out")
	flag.IntVar(&period, "p", 0, "Process time duration")
	flag.StringVar(&method, "c", "GET", "Method to use for the request")
}

// Hook function to allow modifications on the link / category before making the request
type before_eat func(data string) (d string)

// Hook function to allow modifications on the data that returned from the request
type after_eat func(data string) (d string)

type Configuration struct {
	Urls      []Url             // List of urls to fetch data for
	Method    string            // GET/POST request
	Headers   map[string]string // Set of header for each request
	Timeout   int64             // Each url request time out in milisecond
	Period    int               // All execution time frame
	BeforeEat before_eat        // Hook function to allow modifications on the link / category before making the request
	AfterEat  after_eat         //Hook function to allow modifications on the data that returned from the request
}

// a custom type that we can use for handling errors and formatting responses
type handler func(w http.ResponseWriter, r *http.Request) (interface{}, *handlerError)

type handlerError struct {
	Error   error
	Message string
	Code    int
}

type Url struct {
	Category string
	Link     string
}

// Set the main configuration based on the data the user provide or default.
func InitConfiguration() *Configuration {
	if urls == nil {
		fmt.Println("You must provide list of urls to eat")
		flag.Usage()
		os.Exit(1)
	}

	if timeout == -1 {
		fmt.Println("You must provide the process timeout")
		flag.Usage()
		os.Exit(1)
	}

	configuration := &Configuration{
		Urls:    make([]Url, 0),
		Method:  "GET",
		Timeout: 0,
		Period:  8000,
	}

	if timeout != 0 && period != 0 {
		timeout := make(chan bool, 1)
		go func() {
			<-time.After(time.Duration(period) * time.Second)
			timeout <- true
		}()

		go func() {
			<-timeout
			syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		}()
	}

	if urls == nil {
		configuration.Urls = append(configuration.Urls, Url{Link: "http://www.google.com", Category: "test"})
	}

	return configuration
}

func redirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) >= 3 {
		return errors.New("stopped after 3 redirects")
	}
	return nil
}

// Custom timout dialer, and the main reason is for avoiding closing the response before getting its body
func timeoutDialler(configuration *Configuration) *http.Client {
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
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
		fmt.Println("before: " + url.Category + "\n")
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
			data, _ := ioutil.ReadAll(resp.Body)
			str := string(data)
			if configuration.AfterEat != nil {
				str = configuration.AfterEat(str)
			}
			fmt.Println("after: " + url.Category + "\n")
			feeds[url.Category] = append(feeds[url.Category], str)
		}(url)
	}
	wg.Wait()
}

func returnData(feeds *map[string][]string) *map[string][]string {
	return feeds
}

func EatIt(configuration *Configuration) map[string][]string {
	var feeds = make(map[string][]string)
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		_ = <-signalChannel
		returnData(&feeds)
		os.Exit(0)
	}()

	flag.Parse()

	if configuration == nil {
		configuration = InitConfiguration()
	}

	goMaxProcs := os.Getenv("GOMAXPROCS")

	if goMaxProcs == "" {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	fetcher(configuration, feeds)
	return feeds
}
