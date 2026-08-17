package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
	"github.com/xqsit94/shelp/internal/config"
	"github.com/xqsit94/shelp/internal/prompt"
)

func configProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage provider profiles",
		Long:  "List, switch, add, rename and remove named provider profiles.",
	}

	cmd.AddCommand(configProfileListCmd())
	cmd.AddCommand(configProfileUseCmd())
	cmd.AddCommand(configProfileAddCmd())
	cmd.AddCommand(configProfileRemoveCmd())
	cmd.AddCommand(configProfileRenameCmd())

	return cmd
}

func configProfileListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the configured profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			file, err := config.LoadFile()
			if err != nil {
				return err
			}

			names := file.Names()
			if len(names) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No profiles saved yet: shelp is using the default profile.")
				fmt.Fprintln(cmd.OutOrStdout(), "Create one with: shelp config profile add <name>")
				return nil
			}

			t := table.New().
				Border(lipgloss.RoundedBorder()).
				BorderStyle(prompt.TableBorderStyle).
				StyleFunc(func(row, col int) lipgloss.Style {
					if col == 0 {
						return prompt.TableLabelStyle
					}
					return prompt.TableValueStyle
				}).
				Headers("Profile", "Model", "AI URL", "API Key", "Active")

			for _, name := range names {
				profile, _ := file.Get(name)

				active := ""
				if name == file.ActiveProfile {
					active = "*"
				}

				t = t.Row(name, profile.Model, profile.AIURL, config.MaskAPIKey(profile.APIKey), active)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out)
			fmt.Fprintln(out, prompt.TitleBoldStyle.Foreground(prompt.ColorPrimary).Render("Profiles"))
			fmt.Fprintln(out, t)
			fmt.Fprintln(out, prompt.ExplanationStyle.Render("  * = active profile"))
			fmt.Fprintln(out)

			return nil
		},
	}
}

func configProfileUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use [name]",
		Short: "Make a profile the active one",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file, err := config.LoadFile()
			if err != nil {
				return err
			}

			name := args[0]
			if _, ok := file.Get(name); !ok {
				return unknownProfile(file, name)
			}

			file.ActiveProfile = name
			if err := config.SaveFile(file); err != nil {
				return err
			}

			prompt.DisplaySuccess(fmt.Sprintf("Profile %q is now active", name))
			return nil
		},
	}
}

func configProfileAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add [name]",
		Short: "Add a profile with the setup wizard",
		Long:  "Ask for the provider URL, API key and model, and store them under a new profile name.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("profile name cannot be empty")
			}

			file, err := config.LoadFile()
			if err != nil {
				return err
			}

			if _, ok := file.Get(name); ok {
				return &ExitError{Code: 1, Err: fmt.Errorf("profile %q already exists", name)}
			}

			if !prompt.IsInteractive() {
				return &ExitError{Code: 1, Err: fmt.Errorf("adding a profile needs a terminal: run shelp --profile %s config set url|key|model instead", name)}
			}

			result := prompt.RunSetupWizard()
			if result.Cancelled {
				return &ExitError{Code: exitCancelled, Err: errors.New("setup cancelled")}
			}
			if result.AIURL == "" || result.APIKey == "" || result.Model == "" {
				return fmt.Errorf("AI URL, API key and model are required")
			}

			file.Set(name, config.Profile{AIURL: result.AIURL, APIKey: result.APIKey, Model: result.Model})
			if err := config.SaveFile(file); err != nil {
				return err
			}

			fmt.Println()
			prompt.DisplaySuccess(fmt.Sprintf("Profile %q saved", name))
			warnInsecureURL(result.AIURL)

			if file.ActiveProfile != name {
				prompt.DisplaySuccess(fmt.Sprintf("Use it with: shelp --profile %s ... or shelp config profile use %s", name, name))
			}

			return nil
		},
	}
}

func configProfileRemoveCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "remove [name]",
		Short: "Remove a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file, err := config.LoadFile()
			if err != nil {
				return err
			}

			name := args[0]
			if _, ok := file.Get(name); !ok {
				return unknownProfile(file, name)
			}

			switch {
			case name == file.ActiveProfile:
				return &ExitError{Code: 1, Err: fmt.Errorf("cannot remove the active profile %q: switch with shelp config profile use <name> first", name)}
			case len(file.Profiles) == 1:
				return &ExitError{Code: 1, Err: fmt.Errorf("cannot remove %q: it is the only profile", name)}
			}

			if !yes {
				if !prompt.IsInteractive() {
					return &ExitError{Code: 1, Err: errors.New("removing a profile needs a terminal to confirm: pass -y")}
				}
				if !prompt.ConfirmYesNoInteractive(fmt.Sprintf("Remove profile %q?", name)) {
					prompt.DisplayWarning("Removal cancelled.")
					return nil
				}
			}

			file.Delete(name)
			if err := config.SaveFile(file); err != nil {
				return err
			}

			prompt.DisplaySuccess(fmt.Sprintf("Profile %q removed", name))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "do not ask for confirmation")

	return cmd
}

func configProfileRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename [old] [new]",
		Short: "Rename a profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			file, err := config.LoadFile()
			if err != nil {
				return err
			}

			oldName, newName := args[0], strings.TrimSpace(args[1])
			if newName == "" {
				return fmt.Errorf("profile name cannot be empty")
			}

			profile, ok := file.Get(oldName)
			if !ok {
				return unknownProfile(file, oldName)
			}
			if _, exists := file.Get(newName); exists {
				return &ExitError{Code: 1, Err: fmt.Errorf("profile %q already exists", newName)}
			}

			file.Delete(oldName)
			file.Set(newName, profile)
			if file.ActiveProfile == oldName {
				file.ActiveProfile = newName
			}

			if err := config.SaveFile(file); err != nil {
				return err
			}

			prompt.DisplaySuccess(fmt.Sprintf("Profile %q renamed to %q", oldName, newName))
			return nil
		},
	}
}

func unknownProfile(file *config.File, name string) error {
	names := file.Names()
	if len(names) == 0 {
		return &ExitError{Code: 1, Err: fmt.Errorf("unknown profile %q: no profiles are configured yet", name)}
	}

	return &ExitError{Code: 1, Err: fmt.Errorf("unknown profile %q (available: %s)", name, strings.Join(names, ", "))}
}
