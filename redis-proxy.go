package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var startTime = time.Now()

var redisPool *redis.Pool
var maxConnections = 10
var maxIdleConnections = 2

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprintf(w, "Golang Redis proxy by Anson Liu\n")
}

func uptimeHandler(w http.ResponseWriter, r *http.Request) {
	//bypass same origin policy
	w.Header().Set("Access-Control-Allow-Origin", "*")

	diff := time.Since(startTime)

	fmt.Fprintf(w, fmt.Sprintf("Uptime:\t%v\n",diff.String()))

	fmt.Println("Uptime requested")
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://github.com/ansonl/", 301)
}

func createJSONOutput(status int, data string) string {
	outputMap := make(map[string]string)

	outputMap["status"] = strconv.Itoa(status)
	outputMap["data"] = data

	output, err := json.Marshal(outputMap)
	if err != nil {
		fmt.Println(err.Error())
		return err.Error()
	}
	return string(output)
}

func performGet(key string) (int, string) {
	c := redisPool.Get()
	defer c.Close()

	getResult, err := redis.String(c.Do("GET", key))
	if err != nil {
		fmt.Printf("GET error: %v", err.Error())
		return -1, getResult
	}

	return 0, getResult
}

func performSet(key string, value string) (int) {
	c := redisPool.Get()
	defer c.Close()

	setResult, err := redis.String(c.Do("SET", key, value))
	if err != nil {
		fmt.Printf("SET error: %v\n", err.Error())
		return -1
	}

	if setResult == "OK" {
		fmt.Printf("SET successful. '%v'\n", setResult)
		return 0
	} else {
		fmt.Printf("SET result was '%v' when SET '%v' '%v'.\n", setResult, key, value)
		return -1
	}
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	//bypass same origin policy
	w.Header().Set("Access-Control-Allow-Origin", "*")

	r.ParseForm()

	var key string

	if len(r.Form["key"]) > 0 {
		key = r.Form["key"][0]

		status, data := performGet(key)

		value := base64.StdEncoding.EncodeToString([]byte(data))

		fmt.Fprintf(w, createJSONOutput(status, value))

	} else {
		fmt.Fprintf(w, createJSONOutput(-1, "Missing key parameter."))
	}
}

func setHandler(w http.ResponseWriter, r *http.Request) {
	//bypass same origin policy
	w.Header().Set("Access-Control-Allow-Origin", "*")

	r.ParseForm()

	var key string
	var value string

	if len(r.Form["key"]) > 0 && len(r.Form["value"]) > 0 {
		key = r.Form["key"][0]
		value = r.Form["value"][0]

		decodedBytes, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			fmt.Println("Base64 decode error:", err)
		}

		value = string(decodedBytes)

		status := performSet(key, value)

		fmt.Fprintf(w, createJSONOutput(status, ""))

	} else {
		fmt.Fprintf(w, createJSONOutput(-1, "Missing key and value parameters."))
	}
}

func server(wg *sync.WaitGroup) {
	//Called by stickify pusher client
	http.HandleFunc("/set", setHandler)
	//Called by stickify viewer client
	http.HandleFunc("/get", getHandler)

	http.HandleFunc("/uptime", uptimeHandler)
	http.HandleFunc("/about", aboutHandler)
	http.HandleFunc("/", rootHandler)
	//http.ListenAndServe(":8080", nil)

	err := http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		panic(err)
	}

	fmt.Println("Server ended on port " + os.Getenv("PORT"))

	wg.Done()
}

func createRedisPool() *redis.Pool {
	pool := redis.NewPool(func() (redis.Conn, error) {
		c, err := redis.DialURL(os.Getenv("REDIS_URL"))

		if err != nil {
			log.Println(err)
			return nil, err
		}

		return c, err
	}, maxIdleConnections)
	pool.TestOnBorrow = func(c redis.Conn, t time.Time) error {
        if time.Since(t) < time.Minute {
            return nil
        }
        _, err := c.Do("PING")
        return err
    }

	pool.MaxActive = maxConnections
	pool.IdleTimeout = time.Second * 10
	return pool
}

func main() {
	//Setup redis connection pool
	redisPool = createRedisPool()

	//start server and wait
	var wg sync.WaitGroup
	wg.Add(1)
	go server(&wg)
	wg.Wait()

	redisPool.Close()
}