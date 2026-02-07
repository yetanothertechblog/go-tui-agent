package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"go-tui/conversation"
	"go-tui/llm"
	"go-tui/tui"
)

func main() {
	resume := flag.Bool("resume", false, "resume a conversation (optionally pass UUID as positional arg)")
	flag.Parse()

	var resumeID string
	if flag.NArg() > 0 {
		resumeID = flag.Arg(0)
	}

	if err := llm.InitAPIKey(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	logDir := filepath.Join(workingDir, "log")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		fmt.Printf("Error creating log dir: %v\n", err)
		os.Exit(1)
	}
	logFile, err := os.OpenFile(filepath.Join(logDir, "debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("starting go-tui")

	convDir := conversation.Dir(workingDir)
	var conv *conversation.Data

	if resumeID != "" {
		// Explicit UUID provided: go run . -resume <uuid>
		path := filepath.Join(convDir, resumeID+".json")
		conv, err = conversation.Load(path)
		if err != nil {
			fmt.Printf("Error loading conversation: %v\n", err)
			os.Exit(1)
		}
		log.Printf("resumed conversation: %s", conv.ID)
	} else if *resume {
		// -resume with no UUID: resume the latest
		latest, err := conversation.LatestInDir(convDir)
		if err != nil {
			fmt.Printf("No conversations to resume.\n")
			os.Exit(1)
		}
		conv, err = conversation.Load(filepath.Join(convDir, latest))
		if err != nil {
			fmt.Printf("Error loading conversation: %v\n", err)
			os.Exit(1)
		}
		log.Printf("resumed latest conversation: %s", conv.ID)
	} else {
		// No flag: auto-resume latest or create new
		latest, err := conversation.LatestInDir(convDir)
		if err == nil {
			conv, err = conversation.Load(filepath.Join(convDir, latest))
			if err != nil {
				fmt.Printf("Error loading conversation: %v\n", err)
				os.Exit(1)
			}
			log.Printf("auto-resumed conversation: %s", conv.ID)
		} else {
			conv = conversation.New()
			log.Printf("new conversation: %s", conv.ID)
		}
	}

	m := tui.New(workingDir, conv)
	p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	m.SetProgram(p)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
