package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/adrienkohlbecker/errors"
	"github.com/spf13/cobra"

	"github.com/adrienkohlbecker/ejson-kms/crypto"
	"github.com/adrienkohlbecker/ejson-kms/kms"
	"github.com/adrienkohlbecker/ejson-kms/model"
	"github.com/adrienkohlbecker/ejson-kms/utils"
)

const docAdd = `
Add a credential to a credentials file.
`

type addCmd struct {
	credsPath   string
	name        string
	description string
	creds       *model.Store
}

func (cmd *addCmd) Cobra() *cobra.Command {

	c := &cobra.Command{
		Use:   "add NAME",
		Short: "Add a credential to a credentials file.",
		Long:  strings.TrimSpace(docAdd),
	}

	c.Flags().StringVar(&cmd.credsPath, "path", ".credentials.json", "The path of the generated file.")
	c.Flags().StringVar(&cmd.description, "description", "", "Description of the credential.")

	return c
}

func init() {
	addCommand(app, &addCmd{})
}

func (cmd *addCmd) Parse(args []string) errors.Error {

	err := utils.ValidCredentialsPath(cmd.credsPath)
	if err != nil {
		return err
	}

	name, err := utils.HasOneArgument(args)
	if err != nil {
		return err
	}
	cmd.name = name

	err = utils.ValidName(cmd.name)
	if err != nil {
		return err
	}

	creds, err := model.Load(cmd.credsPath)
	if err != nil {
		return errors.WrapPrefix(err, "Unable to load JSON", 0)
	}
	cmd.creds = creds

	if cmd.creds.Contains(cmd.name) {
		return errors.Errorf("A credential with the same name already exists. Use the `rotate` command")
	}

	return nil
}

func (cmd *addCmd) Execute(args []string) errors.Error {

	plaintext, err := utils.ReadFromFile(os.Stdin)
	if err != nil {
		return err
	}

	client, err := kms.NewClient()
	if err != nil {
		return err
	}

	fmt.Printf("KMS: Encrypting plaintext for %s\n", cmd.name)

	now := time.Now().UTC().Truncate(time.Second)
	cipher := crypto.NewCipher(client, cmd.creds.KMSKeyID, cmd.creds.EncryptionContext)

	ciphertext, err := cipher.Encrypt(plaintext)
	if err != nil {
		return errors.WrapPrefix(err, "Unable to encrypt credential", 0)
	}

	cred := model.Credential{
		Name:        cmd.name,
		Description: cmd.description,
		AddedAt:     now,
		RotatedAt:   nil,
		Ciphertext:  ciphertext,
	}

	cmd.creds.Credentials = append(cmd.creds.Credentials, cred)

	err = cmd.creds.Save(cmd.credsPath)
	if err != nil {
		return errors.WrapPrefix(err, "Unable to save JSON", 0)
	}

	fmt.Printf("Exported new credentials file at: %s\n", cmd.credsPath)

	return nil
}
