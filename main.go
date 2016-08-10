package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"flag"
	"github.com/robfig/cron"
)

const CHANNELS_FILE = "channels.json"
const HTTP_ADDRESS = "http://example.com/"
const FOLDER_TIME_FORMAT = "20060102"
var backupAtStart = flag.Bool("b", false, "Backup at startup")
var channels	Container

type Container struct {
	Channels []struct {
		ID string `json:"id"`
		ApiKey string `json:"api_key"`
	}
}

func errorHandling(err error){
	fmt.Println("ERROR:", err)
}

func  unmarshalJson(data []byte, container Container) (Container){
	err := json.Unmarshal(data, &container)
	if err != nil {
		errorHandling(err)
	}
	return container
}

func downloadFromUrl(url string, path string, fileName string) {
	fmt.Println("Downloading", url, "to", fileName)
	// TODO: check file existence first with io.IsExist
	output, err := os.Create(filepath.Join(path, fileName))
	if err != nil {
		errorHandling(err)
		return
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		errorHandling(err)
		return
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		errorHandling(err)
		return
	}

	fmt.Println(n, "bytes downloaded.")
}

func parseChannelsFile() {
	fmt.Println("Parsing ", CHANNELS_FILE, " file...")
	data, err := ioutil.ReadFile(CHANNELS_FILE)
	if err != nil {
		errorHandling(err)
		return
	}
	channels = unmarshalJson(data, channels)
	if err != nil {
                errorHandling(err)
        	return
	}
	fmt.Println("Parsed ", CHANNELS_FILE, " successfully!")
}

func downloadChannels() {
	folderName := "BAK_" + time.Now().Format(FOLDER_TIME_FORMAT)
	fmt.Println("Creating ", folderName, "...")
	err := os.Mkdir(folderName, 0777)
	if err != nil {
		errorHandling(err)
		return
	}
	fmt.Println("Created ", folderName, " successfully!")
	for _,element := range channels.Channels {
		downloadUrl := HTTP_ADDRESS + "stream/channels/" + element.ID + "/feeds?api_key=" + element.ApiKey
		downloadFromUrl(downloadUrl, folderName, element.ID + ".cvs")
	}
}
func main() {
	flag.Parse()
	fmt.Println("Bakbot started successfully!")
	parseChannelsFile()
	c := cron.New()
	c.AddFunc("@daily", downloadChannels)
	c.Start()
	if(*backupAtStart) {
		downloadChannels()
	}
	select {}
}
