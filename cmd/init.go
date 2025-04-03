/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/Hapaa16/lit/config"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		dir, err := os.Getwd()

		if err != nil {
			fmt.Println("Lit error: ", err)
			os.Exit(1)
		}

		path := dir + "/" + config.InitDirName

		err = os.Mkdir(path, 0755)

		if err != nil {
			fmt.Println("Lit error: ", err)
			os.Exit(1)
		}

		for _, d := range config.DefaultDirs {
			err = os.Mkdir(path+"/"+d, 0755)

			if err != nil {
				fmt.Println("Lit error: ", err)
				os.Exit(1)
			}
		}

		f, err := os.Create(path + "/HEAD")

		if err != nil {
			fmt.Println("Lit error: ", err)
			os.Exit(1)
		}

		f.Write([]byte(config.HeadData))
		
		f.Close()

		fmt.Println("LIT Repository created in ", path, "🔥")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
