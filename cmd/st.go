/*
Copyright © 2023 Victor Guidi <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func ReadDir(dirname string) ([]os.FileInfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Readdir(-1)
}

func captureStdout(f func()) string {
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w

	f()

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = stdout
	return string(out)
}

type Language struct {
	name        string
	probability float64
	extensions  []string
}

func estimateProjectType(files map[string]int) string {

	//TODO Add separation for when the project might be a mix of languages, for example a frontend folder using React and a backend folder using Go

	// Currently Checking for five type of projects:
	// 1. Web project
	// 2. Go project
	// 3. C++ project
	// 4. Python project
	// 5. Rust project

	// 1. Aggregate the number of files and find the probability of being one of the above projects types
	// If the probability is high enough, we will return the project type

	var languages = []Language{
		{
			name:        "Web",
			probability: 0.0,
			extensions:  []string{"html", "css", "js", "jsx", "ts", "tsx", "json", "xml", "yml", "yaml", "md", "txt", "csv", "svg", "png", "jpg", "jpeg", "gif", "ico", "webp", "mp4", "webm", "asm"},
		},
		{
			name:        "Go",
			probability: 0.0,
			extensions:  []string{"go", "mod", "sum"},
		},
		{
			name:        "C++",
			probability: 0.0,
			extensions:  []string{"cpp", "h", "out", "c"},
		},
		{
			name:        "Python",
			probability: 0.0,
			extensions:  []string{"py", "pyc", "pyd", "pyo", "pyw", "pyz", "pyi", "pyc", "pyd", "pyo", "pyw", "pyz", "pyi"},
		},
		{
			name:        "Rust",
			probability: 0.0,
			extensions:  []string{"rs", "toml"},
		},
	}

	// We will iterate over the files and check if the extension is in the list of extensions for each language
	for key, value := range files {
		for i := 0; i < len(languages); i++ {
			for j := 0; j < len(languages[i].extensions); j++ {
				if key == languages[i].extensions[j] {
					languages[i].probability += float64(value) / float64(len(files))
				}
			}
		}
	}

	type LanguageProbability struct {
		name        string
		probability float64
	}

	var languageProbability = LanguageProbability{
		name:        "",
		probability: 0.0,
	}

	// We will iterate over the languages and check if the probability is high enough, if is high enough we will update it to the LanguageProbability struct
	for i := 0; i < len(languages); i++ {
		if languages[i].probability > languageProbability.probability {
			languageProbability.name = languages[i].name
			languageProbability.probability = languages[i].probability
		}
	}

	if languageProbability.probability > 0.0 {
		return `The project probably is a ` + languageProbability.name + ` project with a weight of approximatly ` + fmt.Sprintf("%.2f", (languageProbability.probability/float64(len(files)))*100) + "%"
	} else {
		return "Unknown"
	}

}

func readDirectory(dir string) []os.FileInfo {
	files, err := ReadDir(dir)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return nil
	}
	allFiles := []os.FileInfo{}

	for _, file := range files {
		if file.IsDir() {
			subdirFiles := readDirectory(filepath.Join(dir, file.Name()))
			allFiles = append(allFiles, subdirFiles...)
		} else {
			allFiles = append(allFiles, file)
		}
	}
	return allFiles
}

// stCmd represents the st command
var stCmd = &cobra.Command{
	Use:   "st",
	Short: "A brief description of your command",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		// We will Parse the directory path from the args and print all the files in it

		// Reading the config file if exists
		// config, _ := cmd.Flags().GetString("config")
		// var yml map[string]interface{}
		// if config != "" {
		// 	c, err := utils.ReadYml(config)
		// 	if err != nil {
		// 		yml = nil
		// 	} else {
		// 		yml = c
		// 	}
		// }
		// fmt.Println(yml)

		fstatus, _ := cmd.Flags().GetBool("fzf")

		if fstatus {
			// using os/exec to call fzf passing the directory as an argument
			dir := args[0]

			filteredFiles := []string{}

			files, err := ReadDir(dir)
			if err != nil {
				fmt.Println("Error reading directory:", err)
			}

			for _, file := range files {
				filteredFiles = append(filteredFiles, file.Name())
			}

			// Output of the fzf command
			output := captureStdout(func() {
				// we execute the fzf command passing the files as an argument
				cmd := exec.Command("fzf", "--reverse")
				cmd.Stdin = strings.NewReader(strings.Join(filteredFiles, "\n"))
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Run()
			})

			// TODO: Add a way to check if the file is a directory or not and if it is, call the function again
			fmt.Println("Selected file:", output)

		} else {
			dir := args[0]
			files := readDirectory(dir)

			// Vector that will hold the type of each file the amount of times it appears
			types := make(map[string]int)

			for _, file := range files {
				// File extension for each file and add it to the map
				// make sure the file has an extension
				if len(strings.Split(file.Name(), ".")) < 2 {
					continue
				}
				extension := strings.Split(file.Name(), ".")[1]
				types[extension]++
			}

			// for key, value := range types {
			// 	fmt.Println(key, ":", value)
			// }
			fmt.Println(estimateProjectType(types))
		}
	},
}

var cfgFile string

func init() {
	rootCmd.AddCommand(stCmd)

	// Here you will define your flags and configuration settings.
	// stCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.st_cli.yaml)")
	// stCmd.MarkFlagRequired("config")

	stCmd.Flags().BoolP("fzf", "f", false, "Open files with fzf")
	stCmd.Flags().BoolP("config", "c", false, "Use a config file")

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// stCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// stCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
