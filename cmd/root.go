/*
Copyright © 2022 Mark Salter

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/mosalter/testlab/vxi11"
	"github.com/spf13/cobra"
)

var (
	progName = "testlab"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   progName,
	Short: "A utility for remote control of test and measurement equipment",
	Long:  "A utility for remote control of test and measurement equipment",

	//	RunE: func(cmd *cobra.Command, args []string) (err error) {
	//		return doRhelBp(args)
	//	},
	// SilenceUsage:  true,
	SilenceErrors: false,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.AddCommand(showPortsCmd)
	rootCmd.AddCommand(testCmd)
}

var showPortsCmd = &cobra.Command{
	Use:           "showports <host>",
	Short:         "Show VXI11 ports for given host",
	SilenceUsage:  true,
	SilenceErrors: true,

	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ports, err := vxi11.FindPorts(args[0])
		if err == nil {
			fmt.Printf("Core: %d, Abort: %d, IRQ: %d\n", ports[0], ports[1], ports[2])
		}
		return err
	},
}

var testCmd = &cobra.Command{
	Use:           "test <host>",
	Short:         "devel-only testing",
	SilenceUsage:  true,
	SilenceErrors: false,

	RunE: func(cmd *cobra.Command, args []string) (err error) {
		return vxi11.DoTest(args)
	},
}
