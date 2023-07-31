// Package keyring provides password store functionality.
package keyring

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/launchrctl/launchr"
)

// ID is a plugin id.
const ID = "keyring"

func init() {
	launchr.RegisterPlugin(&Plugin{})
}

// Plugin is launchr plugin providing web ui.
type Plugin struct {
	k Keyring
}

// PluginInfo implements launchr.Plugin interface.
func (p *Plugin) PluginInfo() launchr.PluginInfo {
	return launchr.PluginInfo{
		ID: ID,
	}
}

// InitApp implements launchr.Plugin interface.
func (p *Plugin) InitApp(app *launchr.App) error {
	m := app.ServiceManager()
	p.k = newKeyringService(app.GetCfgDir())
	m.Add(ID, p.k)
	return nil
}

var passphrase string

// CobraAddCommands implements launchr.CobraPlugin interface to provide web functionality.
func (p *Plugin) CobraAddCommands(rootCmd *cobra.Command) error {
	var creds CredentialsItem
	var loginCmd = &cobra.Command{
		Use:   "login",
		Short: "Logs in to services like git, docker, etc.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Don't show usage help on a runtime error.
			cmd.SilenceUsage = true
			return login(p.k, creds)
		},
	}
	var logoutCmd = &cobra.Command{
		Use:   "logout",
		Short: "Logs out from a service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Don't show usage help on a runtime error.
			cmd.SilenceUsage = true
			return logout(p.k, args[0])
		},
	}

	// Credentials flags
	loginCmd.Flags().StringVarP(&creds.URL, "url", "", "", "URL")
	loginCmd.Flags().StringVarP(&creds.Username, "username", "", "", "Username")
	loginCmd.Flags().StringVarP(&creds.Password, "password", "", "", "Password")
	// Passphrase flags
	rootCmd.PersistentFlags().StringVarP(&passphrase, "keyring-passphrase", "", "", "Passphrase for keyring encryption/decryption")
	// Command flags.
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	return nil
}

func login(k Keyring, creds CredentialsItem) error {
	// Ask for login elements.
	err := withTerminal(func(in, out *os.File) error {
		return credentialsFromTty(&creds, in, out)
	})
	if err != nil {
		return err
	}

	err = k.AddItem(creds)
	if err != nil {
		return err
	}
	return k.Save()
}

func credentialsFromTty(creds *CredentialsItem, in *os.File, out *os.File) error {
	reader := bufio.NewReader(in)

	if creds.URL == "" {
		fmt.Fprint(out, "URL: ")
		url, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		creds.URL = strings.TrimSpace(url)
	}

	if creds.Username == "" {
		fmt.Fprint(out, "Username: ")
		username, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		creds.Username = strings.TrimSpace(username)
	}

	if creds.Password == "" {
		fmt.Fprint(out, "Password: ")
		bytePassword, err := term.ReadPassword(int(in.Fd()))
		fmt.Fprint(out, "\n")
		if err != nil {
			return err
		}
		creds.Password = strings.TrimSpace(string(bytePassword))
	}
	return nil
}

func logout(k Keyring, url string) error {
	err := k.RemoveItem(url)
	if err != nil {
		return err
	}
	return k.Save()
}
