=======
go2eat
======

Helper library to Consume json data from several urls in GO(golang).

<h1>
    <a name="introduction" class="anchor" href="#introduction"><span class="octicon octicon-link"></span></a>Introduction</h1>

<p>I created this project for two reasons: </p>

<ul>
<li>1. kick starter for me to deep dive into the fasinated world of go</li>

<li>2. We needed fast solution to consume a lot of api data from different providers and return them by pre-defined category
and return them fast (for online manipulations). I first build the same solution with node js using Asyn.Waterfall, but,
had throughput issues and the bunch of callback hell, which make the code very hard to read and follow (even when I used coffee-script)
</li>
</ul>

<h1>
    <a name="usage" class="anchor" href="#usage"><span class="octicon octicon-link"></span></a>Usage</h1>
<ol>
    <li> go get github.com/may215/go2eat </li>
    <li> If you want to configure the urls outside of the code, then you can create list.conf file that contain the json file (you can see the example) </li>
    <li> Follow the configuration instructions and change them according to your needs </li>
    <li>
        <p>go run go2eat.go</p>
    </li>
    <li>
        <p>Working Example:</p>
        <p><pre><code>
            package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/may215/go2eat"
)

type handlerError struct {
	Error   error
	Message string
	Code    int
}

type handler func(w http.ResponseWriter, r *http.Request) (interface{}, *handlerError)

func (fn handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	response, err := fn(w, r)

	if err != nil {
		log.Printf("ERROR: %v\n", err.Error)
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Message), err.Code)
		return
	}
	if response == nil {
		log.Printf("ERROR: response from method is nil\n")
		http.Error(w, "Internal server error. Check the logs.", http.StatusInternalServerError)
		return
	}

	// turn the response into JSON
	bytes, e := json.Marshal(response)
	if e != nil {
		http.Error(w, "Error marshalling JSON", http.StatusInternalServerError)
		return
	}

	// send the response and log
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
	log.Printf("%s %s %s %d", r.RemoteAddr, r.Method, r.URL, 200)
}

func test_before_func(d string) string {
	return d
}

func test_after_func(d string) string {
	return d
}

func getDataPreDefinedUrls(w http.ResponseWriter, r *http.Request) (interface{}, *handlerError) {
	start := time.Now().UnixNano()
	var urls = make([]go2eat.Url, 0)
	urls = append(urls, go2eat.Url{Category: "Popular", Link: "http://www.vice.com/api/getmostpopular/0"})
	urls = append(urls, go2eat.Url{Category: "video", Link: "http://www.vice.com/api/getlatest/video/0"})
	urls = append(urls, go2eat.Url{Category: "Latest", Link: "http://www.vice.com/api/getlatest/0"})
	urls = append(urls, go2eat.Url{Category: "dos-and-donts", Link: "http://www.vice.com/api/getlatest/dos-and-donts"})
	urls = append(urls, go2eat.Url{Category: "News", Link: "http://www.vice.com/api/getlatest/category/News"})
	urls = append(urls, go2eat.Url{Category: "Music", Link: "http://www.vice.com/api/getlatest/category/Music"})
	urls = append(urls, go2eat.Url{Category: "Fashion", Link: "http://www.vice.com/api/getlatest/category/Fashion"})
	urls = append(urls, go2eat.Url{Category: "Travel", Link: "http://www.vice.com/api/getlatest/category/Travel"})
	urls = append(urls, go2eat.Url{Category: "Sports", Link: "http://www.vice.com/api/getlatest/category/Sports"})

	var headers = make(map[string]string)
	headers["User-Agent"] = "Mozilla/5.0 (Windows; U; Windows NT 5.1; en-US) AppleWebKit/525.13 (KHTML, like Gecko) Chrome/0.2.149.29 Safari/525.13"

	configuration := &go2eat.Configuration{
		Urls:      urls,
		Method:    "GET",
		Timeout:   6000,
		Period:    8000,
		BeforeEat: test_before_func,
		AfterEat:  test_after_func,
		FilePath:  "./list.conf",
	}

	data, err := go2eat.EatIt(configuration)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Eating new data ....")
	fmt.Println("Eating took: ", time.Now().UnixNano()-start)
	return data, nil
}

func getDataUsingFileData(w http.ResponseWriter, r *http.Request) (interface{}, *handlerError) {
	start := time.Now().UnixNano()

	var headers = make(map[string]string)
	headers["User-Agent"] = "Mozilla/5.0 (Windows; U; Windows NT 5.1; en-US) AppleWebKit/525.13 (KHTML, like Gecko) Chrome/0.2.149.29 Safari/525.13"

	configuration := &go2eat.Configuration{
		Urls:      nil,
		Method:    "GET",
		Timeout:   6000,
		Period:    8000,
		BeforeEat: test_before_func,
		AfterEat:  test_after_func,
		FilePath:  "./list.conf",
	}

	data, err := go2eat.EatIt(configuration)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Eating new data ....")
	fmt.Println("Eating took: ", time.Now().UnixNano()-start)
	return data, nil
}

func main() {
	port := flag.Int("port", 9090, "port to serve on")

	// setup routes
	router := mux.NewRouter()
	router.Handle("/getDataPreDefinedUrls", handler(getDataPreDefinedUrls)).Methods("GET")
	router.Handle("/getDataUsingFileData", handler(getDataUsingFileData)).Methods("GET")
	http.Handle("/", router)

	log.Printf("Running on port %d\n", *port)

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	err := http.ListenAndServe(addr, nil)
	fmt.Println(err.Error())
}

        </code></pre></p>
    </li>
</ol>

<h1>
    <a name="license" class="anchor" href="#license"><span class="octicon octicon-link"></span></a>License</h1>

<p>Licensed under the New BSD License.</p>

<h1>
    <a name="author" class="anchor" href="#author"><span class="octicon octicon-link"></span></a>Author</h1>

<p>Meir Shamay (<a href="https://www.twitter.com/meir_shamay">@meir_shamay</a>)</p>
