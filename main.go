package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type (
	jsonResponse struct {
		Destiny []string `json:"destiny"`
		Twitch  []string `json:"twitch"`
	}
)

const (
	jsonEndpoint = "https://raw.githubusercontent.com/destinygg/chat-gui/master/assets/emotes.json"
	imgDirectory = "https://raw.githubusercontent.com/destinygg/chat-gui/master/assets/emotes/emoticons/"
	numOfWorkers = 10
)

var (
	directory string
	lowercase bool
)

func init() {
	flag.StringVar(&directory, "d", "", "directory to save images to")
	flag.BoolVar(&lowercase, "lowercase", false, "save images wiht lowercase name")
}

func main() {
	flag.Parse()
	if directory == "" {
		log.Fatalln("please provide a directory to save the images to")
	}

	emotes := getEmoteList()

	emoteChan := make(chan string)

	var wg sync.WaitGroup
	for i := 0; i <= numOfWorkers; i++ {
		wg.Add(1)
		go func() {
			for {
				emote, ok := <-emoteChan
				if !ok {
					wg.Done()
					return
				}
				url := fmt.Sprintf("%s/%s.png", imgDirectory, emote)

				if lowercase {
					emote = strings.ToLower(emote)
				}

				file := filepath.Join(directory, fmt.Sprintf("%s.png", emote))
				downloadImage(url, file)
			}
		}()
	}

	for _, emote := range emotes {
		emoteChan <- emote
	}
	close(emoteChan)
	wg.Wait()
}

func getEmoteList() []string {
	var emotes = make([]string, 0)

	resp, err := http.Get(jsonEndpoint)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Json endpoint returned with status code %d\n", resp.StatusCode)
	}

	var jr jsonResponse
	err = json.NewDecoder(resp.Body).Decode(&jr)
	if err != nil {
		log.Fatalln(err)
	}

	emotes = append(emotes, jr.Destiny...)
	emotes = append(emotes, jr.Twitch...)
	return emotes
}

func downloadImage(url string, file string) {
	if _, err := os.Stat(file); os.IsExist(err) {
		err = os.Remove(file)
		if err != nil {
			log.Printf("File %s already existed but unable to be deleted: %s\n", file, err.Error())
			return
		}
	}

	output, err := os.Create(file)
	if err != nil {
		log.Printf("Error creating file %s\n", file)
		return
	}
	defer output.Close()

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error requesting url %s\n", url)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Request to url %s returned with status code %d\n", url, resp.StatusCode)
		return
	}

	_, err = io.Copy(output, resp.Body)
	if err != nil {
		log.Printf("Error writing to file %s\n", file)
		return
	}
}
