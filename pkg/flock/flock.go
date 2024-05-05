package flock

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"slices"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/mattn/go-zglob"
	"github.com/pelletier/go-toml/v2"
)

type secretDependency struct {
	Name   string
	Source struct {
		App  string
		Name string
	}
	value      string
	alreadySet bool
}

type flyToml struct {
	path    string
	content string
	App     string
	Flock   struct {
		Dependencies struct {
			Apps []struct {
				Name string
			}
			Secrets []secretDependency
		}
	}
	deployed bool
}

func buildFlyTomls(globs []string, envFiles []string) (flyTomls []flyToml, err error) {

	// Load env files
	for _, envFile := range envFiles {
		if _, err := os.Stat(envFile); os.IsNotExist(err) {
			continue
		}
		err = godotenv.Load(envFile)
		if err != nil {
			return nil, fmt.Errorf("error loading '%v' env file: %v", envFile, err)
		}
	}

	// Read and insert env vars into fly.toml files
	newEnvVars := map[string]string{}
	for _, glob := range globs {
		filenames, err := zglob.Glob(glob)
		if err != nil {
			return nil, fmt.Errorf("error serching for glob pattern '%v': %v", glob, err)
		}
		for _, filename := range filenames {
			flyTomlTemplate, err := os.ReadFile(filename)
			if err != nil {
				return nil, fmt.Errorf("error reading fly.toml file '%v': %v", filename, err)
			}
			content := os.Expand(string(flyTomlTemplate), func(key string) string {
				if value, exists := os.LookupEnv(key); exists {
					return value
				} else if value, exists := newEnvVars[key]; exists {
					return value
				} else {
					fmt.Printf("Variable %v=", key)
					fmt.Scanln(&value)
					newEnvVars[key] = value
					return value
				}
			})
			ft := flyToml{path: filename, content: content}
			err = toml.Unmarshal([]byte(content), &ft)
			if err != nil {
				return nil, fmt.Errorf("error unmarshalling fly.toml file '%v': %v", filename, err)
			}
			flyTomls = append(flyTomls, ft)
		}
	}

	// Append new env vars to first env file
	if len(newEnvVars) > 0 {
		firstEnvFile, err := os.OpenFile(envFiles[0], os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("error opening first env file '%v': %v", envFiles[0], err)
		}
		defer firstEnvFile.Close()
		newEnvFileContent, err := godotenv.Marshal(newEnvVars)
		if err != nil {
			return nil, fmt.Errorf("error marshalling new env vars (%v): %v", newEnvVars, err)
		}
		_, err = firstEnvFile.Write([]byte(newEnvFileContent))
		if err != nil {
			return nil, fmt.Errorf("error writing new env vars to first env file '%v': %v", envFiles[0], err)
		}
	}
	return flyTomls, nil
}

func findFlyTomlByApp(tomls []flyToml, app string) (*flyToml, error) {
	idx := slices.IndexFunc(tomls, func(ft flyToml) bool { return ft.App == app })
	if idx == -1 {
		return nil, fmt.Errorf("app '%v' not found", app)
	}
	return &tomls[idx], nil
}

func Up(globs []string, envFiles []string, org string) (err error) {
	flyTomls, err := buildFlyTomls(globs, envFiles)
	if err != nil {
		return fmt.Errorf("error building fly.toml files: %v", err)
	}

	// list existing apps
	listCmd := exec.Command("fly", "apps", "list", "--org", org, "--json")
	listCmdOut, err := listCmd.Output()
	if err != nil {
		return fmt.Errorf("error listing apps: %v", err)
	}
	type flyApp struct {
		Name string
	}
	var existingApps []flyApp
	json.Unmarshal(listCmdOut, &existingApps)

	// Create apps if needed
	for _, ft := range flyTomls {
		if slices.ContainsFunc(existingApps, func(a flyApp) bool { return a.Name == ft.App }) {
			fmt.Printf("App %v already exists, skipping creation\n", ft.App)
			continue
		}
		fmt.Printf("Creating app %v...\n", ft.App)
		createAppCmd := exec.Command("fly", "apps", "create", ft.App, "--org", org)
		createAppCmd.Stdout = os.Stdout
		createAppCmd.Stderr = os.Stderr
		createAppCmd.Stdin = os.Stdin
		err = createAppCmd.Run()
		if err != nil {
			return fmt.Errorf("error creating app '%v': %v", ft.App, err)
		}
	}

	// Set secrets – step 1: ask for values
	for _, ft := range flyTomls {
		// list existing secrets
		listCmd := exec.Command("fly", "secrets", "list", "--app", ft.App, "--json")
		listCmdOut, err := listCmd.Output()
		if err != nil {
			return fmt.Errorf("error listing secrets for app %v: %v", ft.App, err)
		}
		type flySecret struct {
			Name string
		}
		var existingSecrets []flySecret
		json.Unmarshal(listCmdOut, &existingSecrets)

		for secretIdx := range ft.Flock.Dependencies.Secrets {
			secret := &ft.Flock.Dependencies.Secrets[secretIdx]
			if slices.ContainsFunc(existingSecrets, func(s flySecret) bool { return s.Name == secret.Name }) {
				secret.alreadySet = true
				continue
			}
			if secret.Source.App == "" {
				if secret.Source.Name != "" {
					return fmt.Errorf(
						"secret '%v' of app '%v' cannot specify its source secret name when it has no source app",
						secret.Name,
						ft.App,
					)
				}
				fmt.Printf("[App: %v] (leave blank for a random UUID) Secret %v=", ft.App, secret.Name)
				fmt.Scanln(&secret.value)
				if secret.value == "" {
					fmt.Printf("Generating random UUID for secret %v of app %v...\n", secret.Name, ft.App)
					secret.value = uuid.New().String()
				}
			}
		}
	}
	// Set secrets – step 2: actually set them
	for _, ft := range flyTomls {
		secretBindings := []string{}
		for secretIdx := range ft.Flock.Dependencies.Secrets {
			secret := &ft.Flock.Dependencies.Secrets[secretIdx]
			if secret.alreadySet {
				fmt.Printf("Secret %v of app %v already set, skipping\n", secret.Name, ft.App)
				continue
			}
			if secret.Source.App != "" {
				sourceAppIdx := slices.IndexFunc(flyTomls, func(ft flyToml) bool { return ft.App == secret.Source.App })
				if sourceAppIdx == -1 {
					return fmt.Errorf("source app '%v' not found for secret '%v' of app '%v'", secret.Source.App, secret.Name, ft.App)
				}
				sourceApp := flyTomls[sourceAppIdx]
				if secret.Source.Name == "" {
					secret.Source.Name = secret.Name
				}
				sourceSecretIdx := slices.IndexFunc(sourceApp.Flock.Dependencies.Secrets, func(s secretDependency) bool { return s.Name == secret.Source.Name })
				if sourceSecretIdx == -1 {
					return fmt.Errorf("source secret '%v' not found in source app '%v' for secret '%v' of app '%v'", secret.Source.Name, secret.Source.App, secret.Name, ft.App)
				}
				sourceSecret := sourceApp.Flock.Dependencies.Secrets[sourceSecretIdx]
				secret.value = sourceSecret.value
			}
			secretBindings = append(secretBindings, fmt.Sprintf("%v=%v", secret.Name, secret.value))
		}
		if len(secretBindings) > 0 {
			fmt.Printf("Creating secrets for app %v: %v...\n", ft.App, secretBindings)
			createSecretCmd := exec.Command("fly", append([]string{"--app", ft.App, "secrets", "set"}, secretBindings...)...)
			createSecretCmd.Stdout = os.Stdout
			createSecretCmd.Stderr = os.Stderr
			createSecretCmd.Stdin = os.Stdin
			err = createSecretCmd.Run()
			if err != nil {
				return fmt.Errorf("error creating secrets for app '%v': %v", ft.App, err)
			}
		} else {
			fmt.Printf("No secrets to set for app %v\n", ft.App)
		}
	}

	// Deploy apps
	for {
		deployedApps := 0
		for ftIdx := range flyTomls {
			ft := &flyTomls[ftIdx]
			if ft.deployed {
				continue
			}

			// Check dependencies
			hasNotDeployedDependency := false
			for _, dep := range ft.Flock.Dependencies.Apps {
				depFt, err := findFlyTomlByApp(flyTomls, dep.Name)
				if err != nil {
					return fmt.Errorf("error finding dependency '%v' for app '%v': %v", dep.Name, ft.App, err)
				}
				if !depFt.deployed {
					hasNotDeployedDependency = true
					break
				}
			}
			if hasNotDeployedDependency {
				continue
			}

			config, err := os.CreateTemp("", "*.fly.toml")
			if err != nil {
				return fmt.Errorf("error creating temporary file with config for app %v: %v", ft.App, err)
			}
			defer os.Remove(config.Name())
			_, err = config.Write([]byte(ft.content))
			if err != nil {
				return fmt.Errorf("error writing config for app %v: %v", ft.App, err)
			}
			fmt.Printf("Deploying app %v...\n", ft.App)
			deployCmd := exec.Command("fly", "deploy", "--config", config.Name())
			deployCmd.Dir = path.Dir(ft.path)
			deployCmd.Stdout = os.Stdout
			deployCmd.Stderr = os.Stderr
			deployCmd.Stdin = os.Stdin
			err = deployCmd.Run()
			if err != nil {
				return fmt.Errorf("error deploying app '%v': %v", ft.App, err)
			}
			ft.deployed = true
			deployedApps++
		}
		if !slices.ContainsFunc(flyTomls, func(ft flyToml) bool { return !ft.deployed }) {
			return // All apps have been deployed!
		}
		if deployedApps == 0 {
			return fmt.Errorf("no apps deployed – stuck in a loop, check your dependencies")
		}
	}
}

func Down(globs []string, envFiles []string) (err error) {
	flyTomls, err := buildFlyTomls(globs, envFiles)
	if err != nil {
		return fmt.Errorf("error building fly.toml files: %v", err)
	}
	for _, ft := range flyTomls {
		fmt.Printf("Deleting app %v...\n", ft.App)
		deleteCmd := exec.Command("fly", "apps", "destroy", ft.App)
		deleteCmd.Stdout = os.Stdout
		deleteCmd.Stderr = os.Stderr
		deleteCmd.Stdin = os.Stdin
		err = deleteCmd.Run()
		if err != nil {
			return fmt.Errorf("error delete app '%v': %v", ft.App, err)
		}
	}
	return
}
