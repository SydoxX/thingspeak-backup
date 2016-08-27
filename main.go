package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"log"
	"path/filepath"
	"time"
	"flag"
	"github.com/robfig/cron"
	"github.com/BurntSushi/toml"
)

const FOLDER_TIME_FORMAT= "20060102"
const API_TIME_FORMAT 	= "2006-01-02 15:04:05"
var USR_DATA_PATH 	= ".usrdata"
var CHANNELS_FILE	= flag.String("C", "channels.json", "custom channels file")
var CONFIG_FILE		= flag.String("c", "config.toml", "custom config file")
var backupAtStart 	= flag.Bool("b", false, "backup at startup")
var verbose 		= flag.Bool("v", false, "enable verbose option")
var HTTP_ADDRESS string
var BAK_FREQUENCY string
var channels	Container
var usrData	UsrDataStruct

type Container struct {
	Channels []struct {
		ID string `json:"id"`
		ApiKey string `json:"api_key"`
	}
}

type UsrDataStruct struct {
	LastBackup time.Time `json:"last_backup"`
}

func (u *UsrDataStruct) setLastBackup(time time.Time) {
	u.LastBackup = time
	writeUsrData()
}

type Config struct {
	HTTP_ADDRESS string
	BAK_FREQUENCY string
}

func errorHandling(err error){
	log.Println("ERROR:", err)
}

func  unmarshalJson(data []byte, container Container) (Container){
	err := json.Unmarshal(data, &container)
	if err != nil {
		errorHandling(err)
	}
	return container
}

func downloadFromUrl(url string, path string, fileName string) {
	log.Println("Downloading", url, "to", fileName)
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

	log.Println(n, "bytes downloaded.")
}

func parseChannelsFile() {
	log.Println("Parsing ", *CHANNELS_FILE, " file...")
	data, err := ioutil.ReadFile(*CHANNELS_FILE)
	if err != nil {
		errorHandling(err)
		return
	}
	channels = unmarshalJson(data, channels)
	if err != nil {
                errorHandling(err)
        	return
	}
	log.Println("Parsed ", *CHANNELS_FILE, " successfully!")
}

func readConfig() {
	_, err := os.Stat(*CONFIG_FILE)
	if err != nil {
		log.Fatal("Config file is missing: ", *CONFIG_FILE)
	}
	var config Config
	if _, err := toml.DecodeFile(*CONFIG_FILE, &config); err != nil {
		log.Fatal(err)
	}
	HTTP_ADDRESS = config.HTTP_ADDRESS
	BAK_FREQUENCY = config.BAK_FREQUENCY
}

func readUsrData() {
	if _, err := os.Stat(USR_DATA_PATH); os.IsNotExist(err) {
		_, err := os.Create(USR_DATA_PATH)
        	if err != nil {
                	log.Fatal("Failed to create the missing .usrdata!")
        	} else {
			log.Println("First time startup: Created .usrdata...")
			usrData.LastBackup = time.Unix(0, 0)
			writeUsrData()
		}
	} else {
		log.Println("Parsing ", USR_DATA_PATH, " file...")
		data, err := ioutil.ReadFile(USR_DATA_PATH)
        	if err != nil {
        	        errorHandling(err)
        	        return
        	}
        	err = json.Unmarshal(data, &usrData)
        	if err != nil {
        	        errorHandling(err)
        	}
        	log.Println("Parsed ", USR_DATA_PATH, " successfully!")
	}
}

func writeUsrData() {
	content,err := json.Marshal(usrData)
	if err != nil {
                errorHandling(err)
        }
	ioutil.WriteFile(USR_DATA_PATH, content, 0777)
}

func setOutputLog() {
	f, err := os.OpenFile("console.log", os.O_APPEND | os.O_CREATE | os.O_RDWR, 0666)
	if err != nil {
		log.Fatal("Error opening file: %v", err)
	}
	log.SetOutput(f)
}

func downloadChannels() {
	timeNow := time.Now()
	folderName := "BAK_" + timeNow.Format(FOLDER_TIME_FORMAT)
	log.Println("Creating ", folderName, "...")
	err := os.Mkdir(folderName, 0777)
	if err != nil {
		errorHandling(err)
		return
	}
	log.Println("Created ", folderName, " successfully!")
	for _,element := range channels.Channels {
		stringStart := usrData.LastBackup.Format(API_TIME_FORMAT)
		stringStart = strings.Replace(stringStart, " ", "%20", -1)
		downloadUrl := HTTP_ADDRESS + "channels/" + element.ID + "/feeds.csv?start=" + stringStart + "&api_key=" + element.ApiKey
		downloadFromUrl(downloadUrl, folderName, element.ID + ".csv")
	}
	usrData.setLastBackup(timeNow)
}

func main() {
	flag.Parse()
	setOutputLog()
	readConfig()
	readUsrData()
	fmt.Println("Bakbot started successfully!")
	parseChannelsFile()
	c := cron.New()
	c.AddFunc(BAK_FREQUENCY, downloadChannels)
	c.Start()
	if(*backupAtStart) {
		downloadChannels()
	}
	select {}
}
