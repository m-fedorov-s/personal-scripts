package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

const PAPER_API_VERSION_URL = "https://api.papermc.io/v2/projects/paper"
const PAPER_API_BUILDS_URL_TEMPLATE = "https://api.papermc.io/v2/projects/paper/versions/%v/builds"
const PAPER_API_JAR_DOWNLOAD_TEMPLATE = "https://api.papermc.io/v2/projects/paper/versions/%v/builds/%v/downloads/%v"

func LoadPaper(dir string) {
	resp, err := http.Get(PAPER_API_VERSION_URL)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var parsed map[string]interface{}
	json.Unmarshal(body, &parsed)
	fmt.Println(dir, parsed)
}

type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

type TimeInterval struct {
	Start Duration `json:"start"` // Time in HH:MM format
	End   Duration `json:"end"`   // Time in HH:MM format
}

type Location time.Location

func (l Location) MarshalJSON() ([]byte, error) {
	tmp := time.Location(l)
	return json.Marshal((&tmp).String())
}

func (l *Location) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		tmp, err := time.LoadLocation(value)
		if err != nil {
			return err
		}
		*l = Location(*tmp)
		return nil
	default:
		return fmt.Errorf("invalid location")
	}
}

type Weekday time.Weekday

func (d Weekday) MarshalText() ([]byte, error) {
	return json.Marshal(time.Weekday(d).String())
}

func (d *Weekday) UnmarshalText(b []byte) error {
	switch string(b) {
	case "Sunday":
		*d = Weekday(time.Sunday)
		return nil
	case "Monday":
		*d = Weekday(time.Monday)
		return nil
	case "Tuesday":
		*d = Weekday(time.Tuesday)
		return nil
	case "Wednesday":
		*d = Weekday(time.Wednesday)
		return nil
	case "Thursday":
		*d = Weekday(time.Thursday)
		return nil
	case "Friday":
		*d = Weekday(time.Friday)
		return nil
	case "Saturday":
		*d = Weekday(time.Saturday)
		return nil
	default:
		return fmt.Errorf("invalid weekday")
	}
}

type Schedule struct {
	Timezone     Location                 `json:"timezone"`
	DaysSchedule map[Weekday]TimeInterval `json:"days_schedule"`
}

// Config represents the configuration with a schedule to restart the process
type Config struct {
	WorkDir      string   `json:"work_dir"`
	WarnBefore   Duration `json:"warn_before"`
	StopSchedule Schedule `json:"schedule"`
}

type Server struct {
	Config      *Config
	CommandName string
	Args        []string
	Cmd         *exec.Cmd
	WaitPipes   sync.WaitGroup
	StreamIn    io.WriteCloser
	StreamOut   io.ReadCloser
	StreamErr   io.ReadCloser
}

func (s *Server) Init(name string, arg ...string) {
	s.CommandName = name
	s.Args = arg
}

func (s *Server) Start() error {
	fmt.Println("Starting process")
	if s.Cmd != nil {
		return fmt.Errorf("Already started")
	}
	s.WaitPipes.Add(2)
	s.Cmd = exec.Command(s.CommandName, s.Args...)
	s.Cmd.Dir = s.Config.WorkDir
	var err error
	s.StreamIn, err = s.Cmd.StdinPipe()
	if err != nil {
		return err
	}
	s.StreamOut, err = s.Cmd.StdoutPipe()
	if err != nil {
		return err
	}
	s.StreamErr, err = s.Cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = s.Cmd.Start()
	if err != nil {
		return err
	}

	go func() {
		defer s.WaitPipes.Done()
		scanner := bufio.NewScanner(s.StreamOut)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	go func() {
		defer s.WaitPipes.Done()
		redColor := "\033[31m"
		resetColor := "\033[0m"

		scanner := bufio.NewScanner(s.StreamErr)
		for scanner.Scan() {
			// Color the stderr output in red
			fmt.Printf("[%vError%v]: %v\n", redColor, resetColor, scanner.Text())
		}
	}()
	return nil
}

func (s *Server) Stop() error {
	s.StreamIn.Write([]byte("stop\n"))
	s.WaitPipes.Wait()
	err := s.Cmd.Wait()
	if err != nil {
		return err
	}
	s.Cmd = nil
	return nil
}

func (s *Server) Run() error {
	s.Start()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()
		if input == "update-paper" {
			err := s.Stop()
			if err != nil {
				panic(err)
			}
			LoadPaper(s.Config.WorkDir)
			err = s.Start()
			if err != nil {
				panic(err)
			}
		} else if input == "stop" {
			break
		} else {
			s.StreamIn.Write([]byte(input + "\n"))
		}
	}
	fmt.Println("Exiting..")
	return s.Stop()
}

// LoadConfig loads the configuration from a JSON file
func LoadConfig(filename string) (Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Config{}, fmt.Errorf("error opening config file: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("error decoding config: %w", err)
	}

	return config, nil
}

func main() {
	configFilePtr := flag.String("config", "config.json", "path to the config file")
	flag.Parse()
	config, err := LoadConfig(*configFilePtr)
	if err != nil {
		log.Fatal(err)
	}
	server := Server{Config: &config}
	server.Init("java", "-Xmx1G", "-jar", "paper.jar", "nogui")
	server.Run()
}
