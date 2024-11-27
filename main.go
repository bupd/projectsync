package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitRepo holds the information about a repository and its remotes
type GitRepo struct {
	Path    string   `json:"path"`
	Remotes []string `json:"remotes"`
	IsBare  bool     `json:"is_bare"`
}

// Function to get remotes from a Git repository
func getGitRemotes(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get remotes for repo %s: %s", repoPath, err)
	}

	var remotes []string
	// Parse the output of 'git remote -v'
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				remotes = append(remotes, parts[1])
			}
		}
	}
	return remotes, nil
}

// Function to check if a directory is a bare Git repository
func isBareRepo(repoPath string) (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--is-bare-repository")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to check if repo is bare: %s", err)
	}
	return strings.TrimSpace(string(output)) == "true", nil
}

// Function to find all Git repositories (including bare repos) within a directory
func findGitRepos(baseDir string) ([]GitRepo, error) {
	var repos []GitRepo

	// Walk through all directories and subdirectories
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the base directory itself
		if path == baseDir {
			return nil
		}

		// Check if it's a Git repository
		if info.IsDir() && filepath.Base(path) == ".git" || filepath.Base(path) == "worktrees" {
			// If it's a Git repository, get remotes and check if it's bare
			remotes, err := getGitRemotes(path)
			if err != nil {
				return nil
			}

			isBare, err := isBareRepo(path)
			if err != nil {
				return nil
			}

			if isBare {
				paths := strings.Split(path, "/")
				// Remove the last part and append ".git"
				path = strings.Join(paths[:len(paths)-1], "/") + "/.git"
			}

			// Add the repository info to the list
			repos = append(repos, GitRepo{
				Path:    path,
				Remotes: remotes,
				IsBare:  isBare,
			})
		}

		// Prevent going deeper into already-processed Git directories
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking the path %v: %v", baseDir, err)
	}

	return repos, nil
}

// Function to save the configuration to a JSON file
func saveConfig(repos []GitRepo, configFilePath string) error {
	file, err := os.Create(configFilePath)
	if err != nil {
		return fmt.Errorf("error creating config file: %s", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(repos)
	if err != nil {
		return fmt.Errorf("error encoding repos to JSON: %s", err)
	}

	fmt.Printf("Config file saved to: %s\n", configFilePath)
	return nil
}

// Function to restore repositories from the configuration file
func restoreRepos(configFilePath string) error {
	// Read the configuration file
	file, err := os.Open(configFilePath)
	if err != nil {
		return fmt.Errorf("error opening config file: %s", err)
	}
	defer file.Close()

	var repos []GitRepo
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&repos)
	if err != nil {
		return fmt.Errorf("error decoding config file: %s", err)
	}

	// Clone each repository and set remotes
	for _, repo := range repos {
		fmt.Printf("Restoring repo: %s\n", repo.Path)

		// change the path for the repository
		paths := strings.Split(repo.Path, "/")
		// Remove the last part and append ".git"
		repo.Path = strings.Join(paths[:len(paths)-1], "/")

		// Clone the repository (bare or non-bare)
		if repo.IsBare {
			cmd := exec.Command("git", "clone", "--bare", repo.Remotes[0], repo.Path)
			err := cmd.Run()
			if err != nil {
				return fmt.Errorf("failed to clone bare repo %s: %s", repo.Path, err)
			}
		} else {
			cmd := exec.Command("git", "clone", repo.Remotes[0], repo.Path)
			err := cmd.Run()
			if err != nil {
				return fmt.Errorf("failed to clone repo %s: %s", repo.Path, err)
			}
		}

		// Add upstream remote if it's not already added
		if len(repo.Remotes) > 1 {
			if repo.IsBare {
				return nil
			}
			for _, remote := range repo.Remotes[1:] {
				log.Println(remote)
				cmd := exec.Command("git", "remote", "add", "upstream", remote)
				cmd.Dir = repo.Path

				err := cmd.Run()
				if err != nil {
					return fmt.Errorf("failed to add upstream remote to repo %s: %s", repo.Path, err)
				}
			}
		}
	}

	fmt.Println("Repositories restored successfully.")
	return nil
}

func main() {
	// Define CLI flags
	var baseDir string
	var configFilePath string
	var backup bool
	var restore bool

	// Define flags for directory, config file, and commands
	flag.StringVar(&baseDir, "dir", ".", "The base directory to scan for repositories.")
	flag.StringVar(&configFilePath, "config", "repos_config.json", "The path to save/load the config file.")
	flag.BoolVar(&backup, "backup", false, "Backup the repositories into a config file.")
	flag.BoolVar(&restore, "restore", false, "Restore repositories from a config file.")
	flag.Parse()

	// Backup command: Find and save Git repositories
	if backup {
		repos, err := findGitRepos(baseDir)
		if err != nil {
			fmt.Println("Error finding repos:", err)
			os.Exit(1)
		}

		err = saveConfig(repos, configFilePath)
		if err != nil {
			fmt.Println("Error saving config file:", err)
			os.Exit(1)
		}
	}

	// Restore command: Clone repos and add remotes
	if restore {
		err := restoreRepos(configFilePath)
		if err != nil {
			fmt.Println("Error restoring repos:", err)
			os.Exit(1)
		}
	}
}
