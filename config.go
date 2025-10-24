package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Api struct {
		Apikey    string
		Baseurl   string
		Agileurl  string
		Email     string
		Projectid string
		Boardid   int
	}
	Git struct {
		Branchbase string
	}
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

	printWarning("No config file found. Let's create one.")
	fmt.Println()

	printPrompt("Enter Jira API Key")
	apikey, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	config.Api.Apikey = strings.TrimSpace(apikey)

	fmt.Printf("%sEnter Jira Base URL %s(e.g., https://yourcompany.atlassian.net)%s:%s ", colorYellow, colorDim, colorYellow, colorReset)
	baseurl, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	config.Api.Baseurl = strings.TrimSpace(baseurl)

	fmt.Printf("%sEnter Jira Agile URL %s(e.g., https://yourcompany.atlassian.net/rest/agile/1.0)%s:%s ", colorYellow, colorDim, colorYellow, colorReset)
	agileurl, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	config.Api.Agileurl = strings.TrimSpace(agileurl)

	printPrompt("Enter Jira Email")
	email, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	config.Api.Email = strings.TrimSpace(email)

	printPrompt("Enter Jira Project ID")
	projectid, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	config.Api.Projectid = strings.TrimSpace(projectid)

	printPrompt("Enter Jira Board ID")
	boardidStr, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	boardid, err := strconv.Atoi(strings.TrimSpace(boardidStr))
	if err != nil {
		return nil, fmt.Errorf("invalid board ID: %v", err)
	}
	config.Api.Boardid = boardid

	fmt.Printf("%sEnter Git Branch Base %s(e.g., main, master, develop)%s:%s ", colorYellow, colorDim, colorYellow, colorReset)
	branchbase, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	config.Git.Branchbase = strings.TrimSpace(branchbase)

	return config, nil
}

func saveConfig(config *Config, filepath string) error {
	// Create directory if it doesn't exist
	dir := filepath[:strings.LastIndex(filepath, "/")]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Create file
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	defer file.Close()

	// Encode config to TOML
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

	// Save to ~/.config/jig/config.toml
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
