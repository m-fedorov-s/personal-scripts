package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const PAPER_API_VERSION_URL = "https://api.papermc.io/v2/projects/paper"
const PAPER_API_BUILDS_URL_TEMPLATE = "https://api.papermc.io/v2/projects/paper/versions/%v/builds"
const PAPER_API_JAR_DOWNLOAD_TEMPLATE = "https://api.papermc.io/v2/projects/paper/versions/%v/builds/%v/downloads/%v"

func LoadFileIfDoesNotExist(url, dir, filename, checksum string) error {
	f, err := os.OpenFile(dir+"/"+filename, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	downloadRes, err := http.Get(url)
	if err != nil {
		panic(err)
		return err
	}
	defer downloadRes.Body.Close()
	_, err = io.Copy(f, downloadRes.Body)
	if err != nil {
		return err
	}
	if checksum == "" {
		return nil
	}
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return err
	}
	if checksum != fmt.Sprintf("%x", h.Sum(nil)) {
		return fmt.Errorf("Sha256 does not match")
	}
	return nil
}

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
	version := parsed["versions"].([]interface{})[len(parsed["versions"].([]interface{}))-1].(string)
	fmt.Println("Chosen version: " + version)
	buildsResp, err := http.Get(fmt.Sprintf(PAPER_API_BUILDS_URL_TEMPLATE, version))
	if err != nil {
		panic(err)
	}
	defer buildsResp.Body.Close()
	body, err = io.ReadAll(buildsResp.Body)
	if err != nil {
		panic(err)
	}
	json.Unmarshal(body, &parsed)
	build := parsed["builds"].([]interface{})[len(parsed["builds"].([]interface{}))-1].(map[string]interface{})
	buildNumber := int(build["build"].(float64))
	filename := build["downloads"].(map[string]interface{})["application"].(map[string]interface{})["name"].(string)
	checksum := build["downloads"].(map[string]interface{})["application"].(map[string]interface{})["sha256"].(string)
	url := fmt.Sprintf(PAPER_API_JAR_DOWNLOAD_TEMPLATE, version, buildNumber, filename)
	err = LoadFileIfDoesNotExist(url, dir, filename, checksum)
	if os.IsExist(err) {
		fmt.Println("Already newest paper build")
	}
	err = os.Remove(dir + "/paper.jar")
	if err != nil {
		panic(err)
	}
	err = os.Symlink(filename, dir+"/paper.jar")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Sucessfuly downloaded %v\n", filename)
}

func BackupFolder(dir string) error {
	bakName := fmt.Sprintf("%v-backup-%v.tar.bz2", dir, time.Now().Format("2006-02-01_15-04_MST"))
	fmt.Printf("Backing up folder %v to %v\n", dir, bakName)
	bakCmd := exec.Command("tar", "-cvjf", bakName, "./"+dir)
	err := bakCmd.Run()
	if err != nil {
		return fmt.Errorf("%v\nstderr: %v", err, err.(*exec.ExitError).Stderr)
	}
	return nil
}

type ListenRequest struct {
	query    string
	accepted chan struct{}
	found    chan string
}

type InnerCmd int

const (
	Backup InnerCmd = iota
	CloseAccess
	OpenAccess
	Warn
)

type Server struct {
	Config        *Config
	runCtx        context.Context
	contextCancel context.CancelFunc
	Cmd           *exec.Cmd
	WaitWorkers   sync.WaitGroup
	requestsPipe  chan ListenRequest
	inputsPipe    chan string
	outputsPipe   chan string
	innerCmds     chan InnerCmd
}

func (s *Server) startIOListeners(ctx context.Context) error {
	streamIn, err := s.Cmd.StdinPipe()
	if err != nil {
		return err
	}
	streamOut, err := s.Cmd.StdoutPipe()
	if err != nil {
		return err
	}
	streamErr, err := s.Cmd.StderrPipe()
	if err != nil {
		return err
	}
	s.WaitWorkers.Add(3)
	s.inputsPipe = make(chan string, 2)
	go func(ctx context.Context, steamIn io.WriteCloser, pipeIn chan string) {
		defer s.WaitWorkers.Done()
		defer streamIn.Close()
		defer fmt.Println("Input writer: done")
		for {
			select {
			case input := <-pipeIn:
				streamIn.Write([]byte(input + "\n"))
			case <-ctx.Done():
				return
			}
		}
	}(ctx, streamIn, s.inputsPipe)

	s.outputsPipe = make(chan string, 2)
	go func(streamOut io.ReadCloser, pipeOut chan string) {
		defer s.WaitWorkers.Done()
		defer close(pipeOut)
		scanner := bufio.NewScanner(streamOut)
		for scanner.Scan() {
			pipeOut <- scanner.Text()
		}
		fmt.Println("Output reader: done")
	}(streamOut, s.outputsPipe)

	go func(streamErr io.ReadCloser) {
		defer s.WaitWorkers.Done()
		redColor := "\033[31m"
		resetColor := "\033[0m"

		scanner := bufio.NewScanner(streamErr)
		for scanner.Scan() {
			// Color the stderr output in red
			fmt.Printf("[%vError%v]: %v\n", redColor, resetColor, scanner.Text())
		}
		fmt.Println("StdErr reader: done")
	}(streamErr)
	return nil
}

func (s *Server) IsStarted() bool {
	return s.runCtx != nil && s.runCtx.Err() != nil
}

func (s *Server) Start(ctx context.Context) error {
	if s.IsStarted() {
		return fmt.Errorf("Already started")
	}
	fmt.Println("Starting process")
	s.Cmd = exec.Command("java", "-Xms"+s.Config.Memory, "-Xmx"+s.Config.Memory, "-XX:+UseG1GC", "-XX:+ParallelRefProcEnabled", "-jar", "paper.jar", "nogui")
	s.Cmd.Dir = s.Config.WorkDir
	cmdCtx, cancel := context.WithCancel(ctx)
	s.runCtx = cmdCtx
	s.contextCancel = cancel
	var err error
	err = s.startIOListeners(s.runCtx)
	if err != nil {
		return err
	}
	err = s.Cmd.Start()
	if err != nil {
		return err
	}
	// Start listening Worker
	s.WaitWorkers.Add(1)
	go func(ctx context.Context) {
		defer s.WaitWorkers.Done()
		defer fmt.Println("Output analyzer: done")
		var reqPtr *ListenRequest
		for {
			if reqPtr != nil {
				select {
				case text, ok := <-s.outputsPipe:
					if !ok {
						return
					}
					if strings.Contains(text, reqPtr.query) {
						reqPtr.found <- text
						reqPtr = nil
					}
					fmt.Println(text)
				case <-ctx.Done():
					return
				}
			} else {
				select {
				case text, ok := <-s.outputsPipe:
					if !ok {
						return
					}
					fmt.Println(text)
				case req := <-s.requestsPipe:
					reqPtr = &req
					reqPtr.accepted <- struct{}{}
				case <-ctx.Done():
					return
				}
			}
		}
	}(cmdCtx)

	// Start scheduling worker
	s.WaitWorkers.Add(1)
	go func(ctx context.Context) {
		defer s.WaitWorkers.Done()
		defer fmt.Println("Scheduler: done")
		timer := time.NewTimer(time.Hour)
		for {
			nextCommand := Backup
			loc := time.Location(s.Config.AccessSchedule.Timezone)
			now := time.Now().In(&loc)
			midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, &loc)
			var nextTime *time.Time
			for _ = range 8 {
				weekday := Weekday(midnight.Weekday())
				schedule, ok := s.Config.AccessSchedule.DaysSchedule[weekday]
				if ok {
					startTime := midnight.Add(schedule.Start.Duration())
					if (nextTime == nil || startTime.Before(*nextTime)) && now.Before(startTime) {
						nextTime = &startTime
						nextCommand = OpenAccess
					}
					endTime := midnight.Add(schedule.End.Duration())
					for _, offset := range s.Config.WarnBefore {
						warnTime := endTime.Add(-time.Duration(offset))
						if (nextTime == nil || warnTime.Before(*nextTime)) && now.Before(warnTime) {
							nextTime = &warnTime
							nextCommand = Warn
						}
					}
					if (nextTime == nil || endTime.Before(*nextTime)) && now.Before(endTime) {
						nextTime = &endTime
						nextCommand = CloseAccess
					}
				} else {
					fmt.Println("No schedule for day %v", time.Weekday(weekday))
				}
				if time.Weekday(weekday) == time.Monday {
					bakTime := midnight.Add(time.Hour)
					if (nextTime == nil || bakTime.Before(*nextTime)) && now.Before(bakTime) {
						nextTime = &bakTime
						nextCommand = Backup
					}
				}
				midnight = midnight.Add(time.Hour * 24)
			}
			if nextTime == nil {
				fmt.Println("Nothing is scheduled for the next week!")
				timer.Reset(time.Hour)
			} else {
				fmt.Printf("Scheduled %v at %v\n", nextCommand, nextTime.Format("2006-01-02 at 15:04 MST"))
				timer.Reset(time.Until(*nextTime))
			}
			select {
			case <-ctx.Done():
				return
			case t := <-timer.C:
				if nextTime != nil {
					fmt.Printf("[%v Scheduler]: sending command %v\n", t.Format("Jan 02 15:04"), nextCommand)
					s.innerCmds <- nextCommand
				}
			}
		}
	}(cmdCtx)

	return nil
}

func (s *Server) Backup() error {
	notify := make(chan struct{})
	find := make(chan string)
	s.requestsPipe <- ListenRequest{query: "Automatic saving is now disabled", accepted: notify, found: find}
	<-notify
	s.inputsPipe <- "save-off"
	<-find
	s.requestsPipe <- ListenRequest{query: "Saved the game", accepted: notify, found: find}
	<-notify
	s.inputsPipe <- "save-all"
	<-find
	err := BackupFolder(s.Config.WorkDir)
	if err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)
	s.requestsPipe <- ListenRequest{query: "Automatic saving is now enabled", accepted: notify, found: find}
	<-notify
	s.inputsPipe <- "save-on"
	<-find
	return nil
}

func (s *Server) Stop() error {
	s.inputsPipe <- "stop"
	err := s.Cmd.Wait()
	if err != nil {
		return err
	}
	fmt.Println("Cmd finished successfuly!")
	s.contextCancel()
	s.WaitWorkers.Wait()
	s.Cmd = nil
	s.runCtx = nil
	s.contextCancel = nil
	return nil
}

func (s *Server) Run() error {
	runCtx, cancelRun := context.WithCancel(context.Background())
	err := s.Start(runCtx)
	if err != nil {
		return err
	}
	defer cancelRun()
	s.innerCmds = make(chan InnerCmd)

	stdIns := make(chan string)
	scanner := bufio.NewScanner(os.Stdin)
	go func() {
		for {
			for scanner.Scan() {
				if runCtx.Err() != nil {
					return
				}
				stdIns <- scanner.Text()
			}
		}
	}()
outer:
	for {
		select {
		case input := <-stdIns:
			{
				switch input {
				case "update":
					{
						err := s.Stop()
						if err != nil {
							panic(err)
						}
						err = BackupFolder(s.Config.WorkDir)
						if err != nil {
							fmt.Printf("Error during back up: %v\n", err)
							panic(err)
						}
						LoadPaper(s.Config.WorkDir)
						err = s.Start(runCtx)
						if err != nil {
							panic(err)
						}
					}
				case "backup":
					{
						err := s.Backup()
						if err != nil {
							fmt.Printf("Error during backup: %v\n", err)
						}
					}
				case "reboot":
					{
						s.Stop()
						time.Sleep(time.Second)
						err := s.Start(runCtx)
						if err != nil {
							panic(err)
						}
					}
				case "stop":
					break outer
				default:
					s.inputsPipe <- input
				}
			}
		case cmd := <-s.innerCmds:
			{
				switch cmd {
				case Backup:
					err := s.Backup()
					if err != nil {
						fmt.Printf("Error during backup: %v\n", err)
					}
				case CloseAccess:
					{
						fmt.Println("Closing server")
						s.inputsPipe <- "say Server is closing now!"
						time.Sleep(time.Second * 5)
						for _, player := range s.Config.Players {
							switch player.Type {
							case Java:
								s.inputsPipe <- fmt.Sprintf("whitelist remove %v", player.Nickname)
								s.inputsPipe <- fmt.Sprintf("kick %v Server is closed", player.Nickname)
							case Bedrock:
								s.inputsPipe <- fmt.Sprintf("fwhitelist remove %v", player.Nickname)
								s.inputsPipe <- fmt.Sprintf("kick .%v Server is closed", player.Nickname)
							}
							time.Sleep(time.Millisecond * 200)
						}
					}
				case OpenAccess:
					{
						fmt.Println("Opening server")
						for _, player := range s.Config.Players {
							switch player.Type {
							case Java:
								s.inputsPipe <- fmt.Sprintf("whitelist add %v", player.Nickname)
							case Bedrock:
								s.inputsPipe <- fmt.Sprintf("fwhitelist add %v", player.Nickname)
							}
							time.Sleep(time.Millisecond * 200)
						}
					}
				case Warn:
					notify := make(chan struct{})
					find := make(chan string)
					s.requestsPipe <- ListenRequest{query: "of a max of", accepted: notify, found: find}
					<-notify
					s.inputsPipe <- "list"
					// There are 0 of a max of ## players online
					playerList := <-find
					if !strings.Contains(playerList, "There are 0 of a max of") {
						s.inputsPipe <- "say Server will close soon"
					} else {
						fmt.Println("Warn not issued")
					}
				}
			}
		case <-runCtx.Done():
			break outer
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
	LoadPaper(config.WorkDir)
	server := Server{Config: &config, requestsPipe: make(chan ListenRequest)}
	err = server.Run()
	if err != nil {
		panic(err)
	}
}
