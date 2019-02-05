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

	dggJSONResponse struct {
		Prefix string                 `json:"prefix"`
		Twitch bool                   `json:"twitch"`
		Image  []dggJSONResponseImage `json:"image"`
	}

	dggJSONResponseImage struct {
		URL  string `json:"url"`
		Name string `json:"name"`
		Mime string `json:"mime"`
	}
)

const (
	jsonEndpoint     = "https://cdn.destiny.gg/4.2.0/emotes/emotes.json"
	bdggJSONEndpoint = "https://raw.githubusercontent.com/BryceMatthes/chat-gui/master/assets/emotes.json"
	bdggImgDirectory = "https://raw.githubusercontent.com/BryceMatthes/chat-gui/master/assets/emotes/emoticons"
	numOfWorkers     = 10
)

var (
	directory string
	lowercase bool
)

func init() {
	flag.StringVar(&directory, "d", "", "directory to save images to")
	flag.BoolVar(&lowercase, "lowercase", false, "save images with lowercase name")
}

func main() {
	flag.Parse()
	if directory == "" {
		log.Fatalln("please provide a directory to save the images to")
	}

	bdggEmotes := getBddgEmoteList()

	emotes := downloadDggEmotes(directory, jsonEndpoint)
	downloadEmotes(directory, bdggImgDirectory, bdggEmotes)
	emotes = append(emotes, bdggEmotes...)
	emotes = removeDupes(emotes)
	writeEmoteFile(directory, emotes)
}

func removeDupes(elements []string) []string {
	// Use map to record duplicates as we find them.
	encountered := map[string]bool{}
	result := []string{}

	for _, v := range elements {
		if encountered[strings.ToLower(v)] == false {
			encountered[strings.ToLower(v)] = true
			result = append(result, strings.ToLower(v))
		}
	}
	return result
}

func downloadEmotes(directory string, emoteDirectory string, emotes []string) {
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
				url := fmt.Sprintf("%s/%s.png", emoteDirectory, emote)

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

func downloadDggEmotes(directory string, endpoint string) []string {
	resp, err := http.Get(jsonEndpoint)

	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Dgg Json endpoint returned with status code %d\n", resp.StatusCode)
	}

	var jr []dggJSONResponse
	err = json.NewDecoder(resp.Body).Decode(&jr)
	if err != nil {
		log.Fatalln(err)
	}

	emoteChan := make(chan dggJSONResponse)
	emotes := make([]string, 0)

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
				url := emote.Image[0].URL

				if lowercase {
					emote.Prefix = strings.ToLower(emote.Prefix)
				}

				if emote.Image[0].Mime != "image/png" {
					log.Printf("Unexpected mime type %s", emote.Image[0].Mime)
				}

				file := filepath.Join(directory, fmt.Sprintf("%s.png", emote.Prefix))
				downloadImage(url, file)
			}
		}()
	}

	for _, emote := range jr {
		if len(emote.Image) > 0 {
			emoteChan <- emote
			emotes = append(emotes, emote.Prefix)
		}
	}
	close(emoteChan)
	wg.Wait()

	return emotes
}

func getBddgEmoteList() []string {
	var emotes = make([]string, 0)

	resp, err := http.Get(bdggJSONEndpoint)
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

func writeEmoteFile(directory string, emotes []string) {
	file := filepath.Join(directory, "emotes.txt")
	if _, err := os.Stat(file); os.IsExist(err) {
		os.Remove(file)
	}

	output, err := os.Create(file)
	if err != nil {
		log.Printf("Error creating file %s\n", file)
		return
	}

	defer output.Close()
	fmt.Fprintf(output, strings.Join(emotes, ","))
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
		os.Remove(file)
		return
	}

	_, err = io.Copy(output, resp.Body)
	if err != nil {
		log.Printf("Error writing to file %s\n", file)
		return
	}
}
