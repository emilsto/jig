package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/emilsto/jig/jira"
)

type JigRC struct {
	ProjectName string `toml:"project_name"`
	ProjectID string `toml:"project_id"`
	BoardName string `toml:"board_name"`
	BoardID int `toml:"board_id"`
}

type Board struct {
	Name string
	ID int
}

type Project struct {
	Name string
	ID string
	Boards []Board
}

type Config struct {
	Api struct {
		Apikey string `toml:"apikey"`
		Baseurl string `toml:"baseurl"`
		Agileurl string `toml:"agileurl"`
		Email string `toml:"email"`
	} `toml:"api"`
	Git struct {
		Branchbase string `toml:"branchbase"`
	} `toml:"git"`
	Projects []Project `toml:"projects"`
}

func findConfig(filename string) string {
	if _, err := os.Stat(filename); err == nil {
		return filename
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		configPath := filepath.Join(exeDir, filename)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	if homeDir, err := os.UserHomeDir(); err == nil {
		configPath := filepath.Join(homeDir, ".config", "jig", filename)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	return ""
}

func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	blob := make([]byte, stat.Size())
	_, err = bufio.NewReader(file).Read(blob)
	if err != nil && err != io.EOF {
		return nil, err
	}

	config := &Config{}
	_, err = toml.Decode(string(blob), config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func promptForConfig() (*Config, error) {
	reader := bufio.NewReader(os.Stdin)
	config := &Config{}

	printWarning("No config file found.")
	fmt.Println()

	printPrompt("Enter Jira API Key")
	apikey, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	config.Api.Apikey = strings.TrimSpace(apikey)

	fmt.Printf("%sEnter Jira Company Name %s(e.g., yourcompany → https://yourcompany.atlassian.net)%s:%s ", colorYellow, colorDim, colorYellow, colorReset)
	companyName, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	companyName = strings.TrimSpace(companyName)
	config.Api.Baseurl = fmt.Sprintf("https://%s.atlassian.net/rest/api/3", companyName)
	config.Api.Agileurl = fmt.Sprintf("https://%s.atlassian.net/rest/agile/1.0", companyName)

	printPrompt("Enter Jira Email")
	email, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	config.Api.Email = strings.TrimSpace(email)

	tempJiraCfg := jira.Config{
		BaseURL:  config.Api.Baseurl,
		AgileURL: config.Api.Agileurl,
		Email:    config.Api.Email,
		APIKey:   config.Api.Apikey,
	}

	tempClient, err := jira.NewClient(tempJiraCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary jira client: %v", err)
	}

	// Fetch available projects
	fmt.Println()
	printInfo("Fetching available projects...")
	jiraProjects, err := tempClient.GetAllProjects(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch projects: %v", err)
	}

	if len(jiraProjects) == 0 {
		return nil, fmt.Errorf("no projects found for this account")
	}

	// Display projects
	fmt.Println()
	printBold("Available Projects:")
	for i, project := range jiraProjects {
		fmt.Printf("  %d. %s (Key: %s, ID: %s)\n", i+1, project.Name, printHighlight(project.Key), project.ID)
	}

	// Select projects
	fmt.Println()
	printPrompt("Select projects (comma-separated numbers, or 'all' for all projects)")
	projectInput, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	projectInput = strings.TrimSpace(projectInput)

	var selectedProjectIndices []int
	if strings.ToLower(projectInput) == "all" {
		for i := range jiraProjects {
			selectedProjectIndices = append(selectedProjectIndices, i)
		}
	} else {
		for _, numStr := range strings.Split(projectInput, ",") {
			num, err := strconv.Atoi(strings.TrimSpace(numStr))
			if err != nil || num < 1 || num > len(jiraProjects) {
				return nil, fmt.Errorf("invalid project selection: %s", numStr)
			}
			selectedProjectIndices = append(selectedProjectIndices, num-1)
		}
	}

	// Fetch boards for selected projects
	config.Projects = []Project{}
	for _, idx := range selectedProjectIndices {
		jiraProject := jiraProjects[idx]

		fmt.Println()
		printInfo("Fetching boards for project %s...", jiraProject.Name)

		jiraBoards, err := tempClient.GetProjectBoards(context.Background(), jiraProject.Key)
		if err != nil {
			printWarning("Failed to fetch boards for %s: %v", jiraProject.Name, err)
			continue
		}

		if len(jiraBoards) == 0 {
			printWarning("No boards found for project %s", jiraProject.Name)
			continue
		}

		fmt.Println()
		printBold("Boards in %s:", jiraProject.Name)
		for i, board := range jiraBoards {
			fmt.Printf("  %d. %s (ID: %d, Type: %s)\n", i+1, board.Name, board.ID, board.Type)
		}

		fmt.Println()
		printPrompt(fmt.Sprintf("Select boards for %s (comma-separated numbers, or 'all')", jiraProject.Name))
		boardInput, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		boardInput = strings.TrimSpace(boardInput)

		var selectedBoards []Board
		if strings.ToLower(boardInput) == "all" {
			for _, jiraBoard := range jiraBoards {
				selectedBoards = append(selectedBoards, Board{
					Name: jiraBoard.Name,
					ID: jiraBoard.ID,
				})
			}
		} else {
			for _, numStr := range strings.Split(boardInput, ",") {
				num, err := strconv.Atoi(strings.TrimSpace(numStr))
				if err != nil || num < 1 || num > len(jiraBoards) {
					return nil, fmt.Errorf("invalid board selection: %s", numStr)
				}
				jiraBoard := jiraBoards[num-1]
				selectedBoards = append(selectedBoards, Board{
					Name: jiraBoard.Name,
					ID: jiraBoard.ID,
				})
			}
		}

		if len(selectedBoards) > 0 {
			config.Projects = append(config.Projects, Project{
				Name: jiraProject.Name,
				ID: jiraProject.Key,
				Boards: selectedBoards,
			})
		}
	}

	if len(config.Projects) == 0 {
		return nil, fmt.Errorf("no projects with boards were selected")
	}

	fmt.Println()
	fmt.Printf("%sEnter Git Branch Base %s(e.g., main, master, develop)%s:%s ", colorYellow, colorDim, colorYellow, colorReset)
	branchbase, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	config.Git.Branchbase = strings.TrimSpace(branchbase)

	return config, nil
}

func saveConfig(config *Config, filepath string) error {
	dir := filepath[:strings.LastIndex(filepath, "/")]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode config: %v", err)
	}

	return nil
}

func getOrCreateConfig(filename string) (*Config, error) {
	configPath := findConfig(filename)

	if configPath != "" {
		return loadConfig(configPath)
	}

	config, err := promptForConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config from user: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	configPath = filepath.Join(homeDir, ".config", "jig", filename)
	if err := saveConfig(config, configPath); err != nil {
		return nil, err
	}

	fmt.Println()
	printSuccess("Config file created at: %s", printHighlight(configPath))
	fmt.Println()
	return config, nil
}

func findJigRC() string {
	currentDir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		jigrcPath := filepath.Join(currentDir, ".jigrc")
		if _, err := os.Stat(jigrcPath); err == nil {
			return jigrcPath
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break
		}
		currentDir = parent
	}

	return ""
}

func loadJigRC(filepath string) (*JigRC, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	blob := make([]byte, stat.Size())
	_, err = bufio.NewReader(file).Read(blob)
	if err != nil && err != io.EOF {
		return nil, err
	}

	jigrc := &JigRC{}
	_, err = toml.Decode(string(blob), jigrc)
	if err != nil {
		return nil, err
	}

	return jigrc, nil
}

func saveJigRC(jigrc *JigRC) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	filepath := filepath.Join(currentDir, ".jigrc")
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create .jigrc file: %v", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(jigrc); err != nil {
		return fmt.Errorf("failed to encode .jigrc: %v", err)
	}

	return nil
}
