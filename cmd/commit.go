/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Hapaa16/lit/utils"
	"github.com/spf13/cobra"
)

var message string

// commitCmd represents the commit command
var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,

	Run: func(cmd *cobra.Command, args []string) {
		if message == "" {
			fmt.Println("Сайн уу, -m флаг ашиглан commit хийнэ үү")
			return
		}

		staging, err := GetStagingFiles()

		if err != nil {
			log.Fatal(err)
		}

		treeSha, err := staging.CreateTree()

		if err != nil {
			log.Fatal(err)
		}

		latestCommit, err := utils.GetLatestCommit()

		if err != nil {
			log.Fatal(err)
		}

		err = utils.CreateCommitObjectWithTree(treeSha, message, []string{latestCommit})

		if err != nil {
			log.Fatal(err)
		}

	},
}

func GetStagingFiles() (utils.IndexJson, error) {
	wd, err := os.Getwd()

	if err != nil {
		return nil, err
	}

	rootRepo := utils.FindRepoRoot(wd)

	// dirPath := wd + "/.lit/objects"
	indexPath := rootRepo + "/.lit/index.json"

	f, err := os.Open(indexPath)

	if err != nil {
		return nil, err
	}

	var indexJson utils.IndexJson

	err = json.NewDecoder(f).Decode(&indexJson)

	if err != nil {
		return nil, err
	}

	return indexJson, nil

}

func init() {
	rootCmd.AddCommand(commitCmd)

	commitCmd.Flags().StringVarP(&message, "message", "m", "", "Commit хийх үеийн мессеж")

}
