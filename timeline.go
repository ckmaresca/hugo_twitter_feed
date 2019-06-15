// Copyright 2012 Arne Roomann-Kurrik
// Copyright 2019 Chris Maresca
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Downloads a user's public timeline and writes each tweet to an individual file.
package main

// Reads as much of a user's last 3200 public Tweets as the Twitter API
// returns, and prints each Tweet to a file.
//
// This example respects rate limiting and will wait until the rate limit
// reset time to finish pulling a timeline.
//
// An out of sync clock can make it appear that the reset has passed and
// cause extra requests.  Use the following to synchronize your time:
//     ntpd -q
// Or (use any NTP server):
//     ntpdate ntp.ubuntu.com
//
// If rate limiting happens, you'll see the executable pause until it
// estimates that the limit has reset.  A more robust implementation would
// use a different approach than just sleeping, but this is a simple example.
//


import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"io/ioutil"
	"io"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"bufio"
)

func LoadCredentials() (client *twittergo.Client, err error) {
	credentials, err := ioutil.ReadFile("CREDENTIALS")
	if err != nil {
		return
	}
	lines := strings.Split(string(credentials), "\n")
	config := &oauth1a.ClientConfig{
		ConsumerKey:    lines[0],
		ConsumerSecret: lines[1],
	}
	user := oauth1a.NewAuthorizedConfig(lines[2], lines[3])
	client = twittergo.NewClient(config, user)
	return
}

func rsl(fn string, n int) (string, error) {
	if n < 1 {
		return "", fmt.Errorf("invalid request: line %d", n)
	}
	f, err := os.Open(fn)
	if err != nil {
		return "", err
	}
	defer f.Close()
	bf := bufio.NewReader(f)
	var line string
	for lnum := 0; lnum < n; lnum++ {
		line, err = bf.ReadString('\n')
		if err == io.EOF {
			switch lnum {
			case 0:
				return "", errors.New("no lines in file")
			case 1:
				return "", errors.New("only 1 line")
			default:
				return "", fmt.Errorf("only %d lines", lnum)
			}
		}
		if err != nil {
			return "", err
		}
	}
	if line == "" {
		return "", fmt.Errorf("line %d empty", n)
	}
	return line, nil
}

type Args struct {
	ScreenName string
	OutputFile string
	SetCount   int
	SetBatch   int
	SetTotal   int
	SinceID    string
}

func parseArgs() *Args {
	a := &Args{}
	flag.StringVar(&a.ScreenName, "screen_name", "twitterapi", "Screen name")
	flag.StringVar(&a.OutputFile, "out", "content/user_timeline", "Output file")
	flag.IntVar(&a.SetCount, "count",100, "Count")
	flag.IntVar(&a.SetBatch, "batch",10, "Batch")
	flag.IntVar(&a.SetTotal, "total",10,"Total")
	flag.StringVar(&a.SinceID,"since","","Since Date")
	flag.Parse()
	return a
}

func main() {
	
	var (
		err     	error
		client  	*twittergo.Client
		req     	*http.Request
		resp    	*twittergo.APIResponse
		args    	*Args
		max_id  	uint64
		tweet_id	uint64
		query   	url.Values
		results 	*twittergo.Timeline
		text    	[]byte
		out_single 	*os.File
		count		int
		last_if		*os.File
		fileLastID	string
	)
	
	if line, err := rsl("content/last_id", 1); err == nil {
		fmt.Printf("1st line [%v]\n",strings.TrimSpace(line))
		fileLastID = strings.TrimSpace(line)
	} else {
		fmt.Println("rsl:", err)
	}
	
	//if fileLastID, err = LoadLastID(); err != nil {
	//	fmt.Printf("LastID file not loaded: %v\n Using defaults from args.\n\n", err)
	//}
		
	args = parseArgs()
	
	if args.SinceID == "" {
		if fileLastID != "" {
			fmt.Printf("Found fileLastID [%v] - setting arg.SinceID \n", fileLastID)
			args.SinceID = fileLastID
		} else {
			fmt.Printf("LastID not set, fetching all available tweets\n")
		}
	}
	
	count = args.SetCount
	
	if client, err = LoadCredentials(); err != nil {
		fmt.Printf("Could not parse CREDENTIALS file: %v\n", err)
		os.Exit(1)
	}
	
	const (
		urltmpl     = "/1.1/statuses/user_timeline.json?%v"
		minwait     = time.Duration(10) * time.Second
	)
	
	query = url.Values{}
	query.Set("count", fmt.Sprintf("%v", count))
	query.Set("screen_name", args.ScreenName)
	if args.SinceID != "" {
		fmt.Printf("args.SinceID is set to [%v], setting query arg \n", args.SinceID)
		query.Set("since_id", args.SinceID)
	}
	total := 0
	
	BatchLoop:
	for {
		if max_id != 0 {
			query.Set("max_id", fmt.Sprintf("%v", max_id))
		}
		endpoint := fmt.Sprintf(urltmpl, query.Encode())
		if req, err = http.NewRequest("GET", endpoint, nil); err != nil {
			fmt.Printf("Could not parse request: %v\n", err)
			os.Exit(1)
		}
		if resp, err = client.SendRequest(req); err != nil {
			fmt.Printf("Could not send request: %v\n", err)
			os.Exit(1)
		}
		results = &twittergo.Timeline{}
		if err = resp.Parse(results); err != nil {
			if rle, ok := err.(twittergo.RateLimitError); ok {
				dur := rle.Reset.Sub(time.Now()) + time.Second
				if dur < minwait {
					// Don't wait less than minwait.
					dur = minwait
				}
				msg := "Rate limited. Reset at %v. Waiting for %v\n"
				fmt.Printf(msg, rle.Reset, dur)
				time.Sleep(dur)
				continue // Retry request.
			} else {
				fmt.Printf("Problem parsing response: %v\n", err)
			}
		}
		batch := len(*results)
		if batch == 0 {
			fmt.Printf("No more results, end of timeline.\n")
			break
		}
		
		for _, tweet := range *results {
			if text, err = json.MarshalIndent(tweet,"","\t"); err != nil {
				fmt.Printf("Could not encode Tweet: %v\n", err)
				os.Exit(1)
			}
			if out_single, err = os.Create(fmt.Sprintf(args.OutputFile + "_%v.json",tweet.Id())); err != nil {
				fmt.Printf("Could not create output file: %v\n", args.OutputFile)
				continue
			}
			out_single.Write(text)
			out_single.Write([]byte("\n"))
			out_single.Close()
			
			max_id = tweet.Id() - 1
			tweet_id = tweet.Id()
			total += 1
			fmt.Printf("Total = %v of %v total set.\n", total, args.SetTotal)
			if total >= args.SetTotal {
				fmt.Printf("Reach set total limit %v, exiting.\n", total)
				break BatchLoop
			}
		}
		fmt.Printf("Got %v Tweets", batch)
		if resp.HasRateLimit() {
			fmt.Printf(", %v calls available", resp.RateLimitRemaining())
		}
		fmt.Printf(", total %v tweets so far of %v set.", total, args.SetTotal)
		fmt.Printf(".\n")
	}
	if last_if, err = os.Create("content/last_id"); err != nil {
		fmt.Printf("Could not create last ID file: last_id")
	} else {
		last_if.WriteString(fmt.Sprintf("%d\n", tweet_id))
		last_if.Close()
	}
	fmt.Printf("--------------------------------------------------------\n")
	fmt.Printf("Wrote %v Tweets to %v_#.json\n", total, args.OutputFile)
}
