/*
Copyright © 2023 Victor Guidi <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/victorguidi/st_cli/utils"
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

func binarySearch(arr []string, x string) int {
	l := 0
	r := len(arr) - 1
	for l <= r {
		m := l + (r-l)/2
		if arr[m] == x {
			return m
		}
		if arr[m] < x {
			l = m + 1
		} else {
			r = m - 1
		}
	}
	return -1
}

type Language struct {
	name        string
	probability float64
	extensions  []string
}

type LanguageProbability struct {
	name        string
	probability float64
}

type Folders struct {
	name        string
	projectType *LanguageProbability
	files       []os.FileInfo
	weight      int
}

func estimateProjectType(files map[string]int) *LanguageProbability {

	//TODO Check if there are any test files, if there are, we will also return the weight of the test files
	//Based on the project structure, we will generate a DOCKERFILE and a docker-compose.yml file suited for the project
	// TODO Add the possibility to parse only for some specific languages given by the user in the config file

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
			sort.Slice(languages[i].extensions, func(i2, j int) bool {
				return languages[i].extensions[i2] < languages[i].extensions[j]
			})
			if binarySearch(languages[i].extensions, key) != -1 {
				// The probability of the language can be maximum 1.0, for example if I have 10 files and 5 of them are .go files, the probability of the project being a Go project is 0.5
				languages[i].probability += float64(value) / float64(len(files))
				if languages[i].probability > 1.0 {
					languages[i].probability = 1.0
				}
			}

		}
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
		// return `The project probably is a ` + languageProbability.name + ` project with a weight of approximatly ` + fmt.Sprintf("%.2f", (languageProbability.probability/float64(len(files)))*100) + "%"
		return &languageProbability
	} else {
		return nil
	}

}

// func readDirectory(dir string) []os.FileInfo {
func readDirectory(dir string, yml *map[string]interface{}, folders *[]Folders) []os.FileInfo {

	files, err := ReadDir(dir)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return nil
	}
	allFiles := []os.FileInfo{}
loop:
	for _, file := range files {
		if file.IsDir() {
			for _, ignore := range (*yml)["ignore"].([]interface{}) {
				if file.Name() == ignore {
					continue loop
				}
			}

			if (*yml)["classfied"].(interface{}).(bool) {
				processSubDirectory := func() {
					subdirFiles := readDirectory(filepath.Join(dir, file.Name()), yml, folders)
					types := make(map[string]int)
					for _, subFile := range subdirFiles {
						ext := filepath.Ext(subFile.Name())
						if ext == "" {
							continue
						}
						types[strings.TrimPrefix(ext, ".")]++
					}
					*folders = append(*folders, Folders{
						name:        file.Name(),
						projectType: estimateProjectType(types),
						weight: func() int {
							var total int
							for _, value := range subdirFiles {
								total += int(value.Size())
							}
							return total
						}(),
					})
				}

				if (*yml)["folders"].([]interface{})[0] != nil {
					for _, folder := range (*yml)["folders"].([]interface{}) {
						switch file.Name() {
						case folder:
							processSubDirectory()
							continue
						default:
							allFiles = append(allFiles, file)
						}
					}
				} else {
					processSubDirectory()
				}
			} else {
				allFiles = append(allFiles, file)
			}
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
		config, _ := cmd.Flags().GetBool("config")
		var yml map[string]interface{}
		if config {
			c, err := utils.ReadYml(args[1])
			if err != nil {
				panic(err)
			} else {
				yml = c
			}
		}

		folders := []Folders{}

		dir := args[0]
		files := readDirectory(dir, &yml, &folders)
		// files := readDirectory(dir)

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
		final := estimateProjectType(types)

		if final != nil {

			fmt.Printf("The project probably is a %s project with a weight of approximatly %.2f%%\n", final.name, (final.probability)*100)

			// print a formated table with the information present in final
			table := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)

			fmt.Fprintln(table, "Folder_Name\tType\tWeight\tTotalSize\t")
			if len(folders) > 0 {
				fmt.Printf("\nHere is a list By Folder:\n\n")
				for _, folder := range folders {
					if folder.projectType == nil {
						fmt.Fprintln(table, folder.name+"\t", "Unknown"+"\t", "Unknown"+"\t", fmt.Sprintf("%.2f", (float64(folder.weight)/1024)/1024)+"MB"+"\t")
						continue
					}
					fmt.Fprintln(table, folder.name+"\t", folder.projectType.name+"\t", fmt.Sprintf("%.1f", (folder.projectType.probability*100))+"%"+"\t"+fmt.Sprintf("%.2f", (float64(folder.weight)/1024)/1024)+"MB"+"\t")
				}
			} else {
				fmt.Fprintln(table, "All Folders", final.name+"\t", fmt.Sprintf("%.1f", (final.probability*100))+"%"+"\t")

			}
			table.Flush()
		}

		// fstatus, _ := cmd.Flags().GetBool("fzf")

		// if fstatus {
		// using os/exec to call fzf passing the directory as an argument
		// dir := args[0]

		// filteredFiles := []string{}

		// files, err := ReadDir(dir)
		// if err != nil {
		// 	fmt.Println("Error reading directory:", err)
		// }

		// for _, file := range files {
		// 	filteredFiles = append(filteredFiles, file.Name())
		// }

		// // Output of the fzf command
		// output := captureStdout(func() {
		// 	// we execute the fzf command passing the files as an argument
		// 	cmd := exec.Command("fzf", "--reverse")
		// 	cmd.Stdin = strings.NewReader(strings.Join(filteredFiles, "\n"))
		// 	cmd.Stdout = os.Stdout
		// 	cmd.Stderr = os.Stderr
		// 	cmd.Run()
		// })

		// // TODO: Add a way to check if the file is a directory or not and if it is, call the function again
		// fmt.Println("Selected file:", output)

		// } else {
		// }
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
