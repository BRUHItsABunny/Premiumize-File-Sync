package main

import (
	"encoding/json"
	"fmt"
	"github.com/BRUHItsABunny/Premiumize-File-Sync/utils"
	premiumize "github.com/BRUHItsABunny/go-premiumize"
	"github.com/BRUHItsABunny/go-premiumize/client"
	"github.com/cornelk/hashmap"
	"github.com/davecgh/go-spew/spew"
	"github.com/dustin/go-humanize"
	"github.com/joho/godotenv"
	"go.uber.org/atomic"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func defaultClient() *client.PremiumizeClient {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Cannot load .env")
		panic(err)
	}
	hClient := http.DefaultClient
	if os.Getenv("TEST_PROXY") == "true" {
		pUrlObj, err := url.Parse(os.Getenv("TEST_PROXY_URL"))
		if err != nil {
			panic(err)
		}
		hClient.Transport = http.DefaultTransport
		hClient.Transport.(*http.Transport).Proxy = http.ProxyURL(pUrlObj)
	}
	return premiumize.GetPremiumizeClient(premiumize.GetPremiumizeAPISession(os.Getenv("PREMIUMIZE_API_KEY")), hClient)
}

func TestPremiumize(t *testing.T) {
	_ = utils.LoadEnv()
	pClient := defaultClient()
	directory := utils.LocateDirectory(pClient, os.Getenv("PREMIUMIZE_TARGET_FOLDER"), true)
	fmt.Println(spew.Sdump(directory))
	fmt.Println("Total size: " + humanize.Bytes(uint64(directory.TotalSize.Load())))
}

func Test_Truncate(t *testing.T) {
	in := "bunnybunbunbunnybunbunbunnybunbunbunnybunbunbunnybunbunbunnybunbun"
	fmt.Println(utils.Truncate(in, 64, 1))
}

func Test_HashmapMarshal(t *testing.T) {
	hm := hashmap.New[string, string]()
	hm.Set("key1", "value1")

	jsonBytes, err := json.Marshal(hm)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(jsonBytes))
}

// go test -v -run Test_ThreadSafety -race -vet=off
func Test_ThreadSafety(t *testing.T) {
	type Test struct {
		Field1 *atomic.String
		Field2 *atomic.Int64
		Field3 *atomic.Time
	}

	data := map[string]*Test{
		"test1": &Test{
			Field1: atomic.NewString("test1"),
			Field2: atomic.NewInt64(0),
			Field3: atomic.NewTime(time.Now()),
		},
		"test2": &Test{
			Field1: atomic.NewString("test2"),
			Field2: atomic.NewInt64(0),
			Field3: atomic.NewTime(time.Now()),
		},
	}

	test1 := data["test1"]
	test2 := data["test2"]
	go func() {
		for i := 0; i < 10000; i++ {
			test1.Field2.Inc()
			test1.Field3.Store(time.Now())
			// fmt.Println("test1")
			time.Sleep(time.Millisecond * time.Duration(250))
		}
	}()

	go func() {
		for a := 0; a < 10000; a++ {
			test2.Field2.Inc()
			test2.Field3.Store(time.Now())
			// fmt.Println("test2")
			time.Sleep(time.Millisecond * time.Duration(500))
		}
	}()

	for i := 0; i < 30; i++ {
		jsonbytes, err := json.Marshal(data)
		if err != nil {
			t.Error()
		}
		fmt.Println(string(jsonbytes))
		time.Sleep(time.Second)
	}
}

func TestTimeMarshal(t *testing.T) {
	type Test struct {
		Field1 time.Time
		Field2 *atomic.Time
	}

	obj := &Test{Field1: time.Now()}
	obj.Field2 = atomic.NewTime(obj.Field1)
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		t.Error(err)
	}

	tmpMap := make(map[string]interface{})
	err = json.Unmarshal(jsonBytes, &tmpMap)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(fmt.Sprintf("JSON:\n%s", string(jsonBytes)))
	if tmpMap["Field1"] != tmpMap["Field2"] { // Field2 is not even a string as of right now
		t.Error("Marshaling not equal")
	}
}

func TestChannelLogic(t *testing.T) {
	wg := &sync.WaitGroup{}
	notification := make(chan struct{}, 6)

	for j := 0; j < 6; j++ {
		notification <- struct{}{}
	}

	wg.Add(1)
	go func() {
		for i := 0; i < 12; i++ {
			<-notification
			fmt.Println(i)
		}
		wg.Done()
	}()

	for j := 0; j < 6; j++ {
		notification <- struct{}{}
		time.Sleep(time.Second)
	}
	wg.Wait()
}

func TestFSLogic(t *testing.T) {
	fName := "bunny/bunbunbun/bunny.bun"
	dName := filepath.Dir(fName)
	err := os.MkdirAll(dName, 0600)
	if err != nil {
		t.Error(err)
	}

	f, err := os.OpenFile(fName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Error(err)
	}
	if f == nil {
		t.Error("file is nil")
	}

}

func TestChannelLogicWorker(t *testing.T) {
	threadCount := 6
	limit := 10
	feedChan := make(chan int, threadCount-1)

	go func() {
		for i := 0; i <= limit; i++ {
			fmt.Println(fmt.Sprintf("Feeding: %d", i))
			feedChan <- i
			fmt.Println(fmt.Sprintf("Fed: %d", i))
			// time.Sleep(time.Second)
		}
	}()

	for {
		fmt.Println(fmt.Sprintf("Waiting for feed"))
		r := <-feedChan
		fmt.Println(fmt.Sprintf("Feed gave: %d", r))
		if r >= limit {
			break
		}
		time.Sleep(time.Second)
	}

	fmt.Println("Done")
}
