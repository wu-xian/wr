package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	gt "github.com/bas24/googletranslatefree"
	"github.com/pelletier/go-toml/v2"

	"github.com/gocarina/gocsv"
)

const FILE = "words.csv"
const ENABLE_TRANS = true

func main() {
	// config := new(Config)
	// config.Load()

	path, _ := os.Executable()
	filePath := filepath.Join(filepath.Dir(path), FILE)
	wordFile, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer wordFile.Close()
	wordMaps, err := gocsv.CSVToMaps(wordFile)
	if err != nil {
		panic(err)
	}
	wordMap := make(map[string]*SaveFile)
	wordHeader := make([]string, 0)
	for _, wordMapOne := range wordMaps {
		word := wordMapOne["word"]
		if len(word) == 0 {
			continue
		}
		wordHeader = append(wordHeader, word)
		count := wordMapOne["count"]
		explain := wordMapOne["explain"]
		times := wordMapOne["times"]
		wordMap[word] = &SaveFile{
			Word:    word,
			Count:   count,
			Explain: explain,
			Times:   times,
		}
	}

	osSignal := make(chan os.Signal, 0)
	signal.Notify(osSignal, os.Kill, os.Interrupt)
	go func() {
		os.Stdin.Seek(0, 0)
		for {
			handler(wordMap, &wordHeader)
		}
	}()

	<-osSignal

	fmt.Println("->saving")
	wordSlice := make([]*SaveFile, 0, len(wordMap))
	for _, header := range wordHeader {
		f, ok := wordMap[header]
		if ok {
			wordSlice = append(wordSlice, f)
		}
	}
	sort.Sort(SaveFiles(wordSlice))
	wordFile.Seek(0, 0)
	err = gocsv.MarshalFile(&wordSlice, wordFile)
	if err != nil {
		fmt.Println("gocsv marshalfile error,err=" + err.Error())
	}
}

func handler(saveFile map[string]*SaveFile, header *[]string) {
	fmt.Print(":")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		fmt.Println("scan false")
		return
	}
	word := scanner.Text()
	var saveFileObj *SaveFile
	f, ok := saveFile[word]
	if !ok {
		saveFileObj = &SaveFile{
			Word:    word,
			Count:   "0",
			Explain: "",
			Times:   "",
		}
		saveFile[word] = saveFileObj
		*header = append(*header, word)
	} else {
		saveFileObj = f
	}

	if len(saveFileObj.Times) == 0 {
		saveFileObj.Times = time.Now().Format("0102150405")
	} else {
		saveFileObj.Times = saveFileObj.Times + "|" + time.Now().Format("0102150405")
	}

	count, err := strconv.Atoi(saveFileObj.Count)
	if err == nil {
		count = count + 1
	} else {
		count = 1
	}
	saveFileObj.Count = strconv.Itoa(count)

	if len(saveFileObj.Explain) == 0 && ENABLE_TRANS {
		txt, err := gt.Translate(saveFileObj.Word, "en", "zh")
		if err != nil {
			fmt.Println("transerr=", err.Error())
		}
		saveFileObj.Explain = txt
	}
	if len(saveFileObj.Explain) != 0 {
		fmt.Println("->" + saveFileObj.Explain)
	}

}

type SaveFile struct {
	Word    string `csv:"word"`
	Count   string `csv:"count"`
	Explain string `csv:"explain"`
	Times   string `csv:"times"`
}
type SaveFiles []*SaveFile

func (s SaveFiles) Len() int {
	return len(s)
}
func (s SaveFiles) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s SaveFiles) Less(i, j int) bool {
	return s[i].Count > s[j].Count
}

type YouDaoConfig struct {
	AppID     string `toml:"app_id"`
	AppSecret string `toml:"app_secret"`
}

type Baiduonfig struct {
	AppID     string `toml:"app_id"`
	AppSecret string `toml:"app_secret"`
}

type Config struct {
	Service string       `toml:"service"`
	YouDao  YouDaoConfig `toml:"youdao"`
	Baidu   Baiduonfig   `toml:"baidu"`
}

func (c *Config) Load() {
	confBytes, err := ioutil.ReadFile("config.toml")
	if err != nil {
		panic(err)
	}
	err = toml.Unmarshal(confBytes, c)
	if err != nil {
		panic(err)
	}
}

type YouDaoRequest struct {
	Q    string `json:"q"`
	From string `json:"from"`
	To   string `json:"to"`
}

type BaiduTransResult struct {
	From        string `json:"from"`
	To          string `json:"to"`
	TransResult []struct {
		Src  string `json:"src"`
		Dst  string `json:"dst"`
		Dict string `json:"dict"`
	} `json:"trans_result"`
	ErrorCode string `json:"error_code"`
}
