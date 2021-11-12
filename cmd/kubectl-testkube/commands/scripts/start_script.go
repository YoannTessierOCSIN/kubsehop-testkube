package scripts

import (
	"fmt"
	"os"
	"time"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

const WatchInterval = 2 * time.Second

var watch bool
var params map[string]string

func NewStartScriptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start",
		Aliases: []string{"run"},
		Short:   "Starts new script",
		Long:    `Starts new script based on Script Custom Resource name, returns results to console`,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			if len(args) == 0 {
				ui.ExitOnError("Invalid arguments", fmt.Errorf("please pass script name to run"))
			}

			scriptID := args[0]

			client, namespace := GetClient(cmd)
			namespacedName := fmt.Sprintf("%s/%s", namespace, scriptID)

			execution, err := client.ExecuteScript(scriptID, namespace, name, params)
			ui.ExitOnError("starting script execution "+namespacedName, err)

			PrintExecutionDetails(execution)

			result := execution.ExecutionResult

			switch true {

			case result.IsQueued():
				ui.Warn("Script queued for execution")

			case result.IsPending():
				ui.Warn("Script execution started")

			case result.IsSuccesful():
				fmt.Println(result.Output)
				duration := execution.EndTime.Sub(execution.StartTime)
				ui.Success("Script execution completed with sucess in " + duration.String())

			case result.IsFailed():
				fmt.Println(result.ErrorMessage)
				ui.Errf("Script execution failed")
			}

			uiShellCommandBlock(execution.Id)

			if watch {
				ui.Info("Watching for changes")

				logs, err := client.Logs(execution.Id)
				ui.ExitOnError("getting logs from exxcutor", err)

				for l := range logs {
					ui.LogLine(l.String())
				}

				for range time.Tick(time.Second) {

					execution, err := client.GetExecution("-", execution.Id)
					ui.ExitOnError("get script execution details", err)

					render := GetExecutionRenderer(cmd)
					err = render.Watch(execution, os.Stdout)
					ui.ExitOnError("watching for changes", err)

					if execution.ExecutionResult.IsCompleted() {
						ui.Info("\nGetting results")
						render.Render(execution, os.Stdout)
						ui.ShellCommand(
							"Use following command to get script execution details",
							"kubectl testkube scripts execution "+execution.Id,
						)
						ui.Warn("Script execution completed in", execution.Duration().String())

						if execution.ExecutionResult.IsFailed() {
							os.Exit(1)
						}
						return
					}
				}

			}

			if execution.ExecutionResult.IsFailed() {
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "execution name, if empty will be autogenerated")
	cmd.Flags().StringToStringVarP(&params, "param", "p", map[string]string{}, "execution envs passed to executor")
	cmd.Flags().BoolVarP(&watch, "watch", "f", false, "watch for changes after start")

	return cmd
}

func uiShellCommandBlock(id string) {
	ui.ShellCommand(
		"Use following command to get script execution details",
		"kubectl testkube scripts execution "+id,
	)
	ui.ShellCommand(
		"or watch script execution until complete",
		"kubectl testkube scripts watch "+id,
	)

	ui.NL()
}
