/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/Hapaa16/lit/config"
	"github.com/Hapaa16/lit/utils"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		wd, err := os.Getwd()

		if err != nil {
			log.Fatal(err)
		}
		repoRoot := utils.FindRepoRoot(wd)

		if repoRoot == "" {
			log.Fatal("Not inside a .lit repo")
		}

		// Get full path of the file being added
		absPath := filepath.Join(wd, args[0])

		// Compute the relative path to the repo root
		relPath, err := filepath.Rel(repoRoot, absPath)

		if err != nil {
			log.Fatal(err)
		}

		blobHash, bString, gitMode, err := utils.FileToBlob(repoRoot + "/" + relPath)

		if err != nil {
			log.Fatal(err)
		}

		dName := blobHash[:2]

		dirPath := path.Join(repoRoot, string(os.PathSeparator), config.InitDirName, string(os.PathSeparator), "objects", dName)

		err = os.MkdirAll(dirPath, 0755)

		if err != nil {
			log.Fatal(err)
		}

		fHash := blobHash[2:]

		blobObjectPath := dirPath + "/" + fHash

		emptyOutput, err := os.Create(blobObjectPath)

		if err != nil {
			log.Fatal(err)
		}

		defer emptyOutput.Close()

		file, err := os.Open(wd + "/" + args[0])

		if err != nil {
			log.Fatal(err)
		}

		defer file.Close()

		w := zlib.NewWriter(emptyOutput)

		w.Write([]byte(bString))

		defer w.Close()

		_, err = io.Copy(w, file)

		if err != nil {
			log.Fatal(err)
		}

		indexPath := path.Join(repoRoot, string(os.PathSeparator), config.InitDirName, "/index.json")

		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			// Data to write

			newBlobIndex := utils.BlobFile{
				Hash: fHash,
				Path: blobObjectPath,
				Mode: gitMode,
			}

			data := map[string]interface{}{
				relPath: newBlobIndex,
			}

			file, _ := os.Create(indexPath)
			defer file.Close()

			json.NewEncoder(file).Encode(data)
			return
		}

		indexFile, err := os.Open(indexPath)

		if err != nil {
			log.Fatal(err)
		}

		defer indexFile.Close()

		var index utils.IndexJson

		json.NewDecoder(indexFile).Decode(&index)

		index[relPath] = utils.BlobFile{
			Hash: fHash,
			Path: blobObjectPath,
			Mode: gitMode,
		}

		indexFile, err = os.OpenFile(indexPath, os.O_WRONLY|os.O_TRUNC, 0755)

		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		defer indexFile.Close()

		if err := json.NewEncoder(indexFile).Encode(index); err != nil {
			fmt.Println("Error writing JSON:", err)
		}

	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
