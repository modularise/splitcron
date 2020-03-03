// This package implements a helper utility to encrypt data in accordance with the requirements for
// storing secrets in GitHub for use in GitHub Actions.

package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jamesruan/sodium"
	"github.com/spf13/cobra"
)

func main() {
	var key, secret, file string

	cmd := cobra.Command{
		Use: os.Args[0],
		RunE: func(_ *cobra.Command, _ []string) error {
			if key == "" {
				return errors.New("the -key flag must be set to a base64-encoded public key")
			} else if (file == "" && secret == "") || (file != "" && secret != "") {
				return errors.New("either the --file or the --secret flag must be set to specify the data to encrypt")
			}

			kb, err := base64.StdEncoding.DecodeString(key)
			if err != nil {
				return fmt.Errorf("failed to decode supplied public key %q as base64 string", key)
			}

			var data []byte
			if secret != "" {
				data = []byte(secret)
			} else if file != "" {
				data, err = ioutil.ReadFile(file)
				if err != nil {
					return fmt.Errorf("failed to read content of data file %q", file)
				}
			}

			enc := sodium.Bytes(data).SealedBox(sodium.BoxPublicKey{Bytes: sodium.Bytes(kb)})

			fmt.Printf("Secret: %s\n", base64.StdEncoding.EncodeToString(enc))
			return nil
		},
	}

	cmd.Flags().StringVar(&file, "file", "", "Path to file to encrypt")
	cmd.Flags().StringVar(&key, "key", "", "Base64-encoded public key to use for the encryption")
	cmd.Flags().StringVar(&secret, "secret", "", "String-value to encrypt")

	if err := cmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
