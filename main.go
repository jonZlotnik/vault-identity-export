package main

import (
	"log"
	"os"

	vault "github.com/hashicorp/vault/api"
)

func main() {
	vaultConfig := vault.DefaultConfig()
	vaultClient, err := vault.NewClient(vaultConfig)
	if err != nill {
		log.Fatalf("Unable to initialize Vault client: %v", err)
	}
	setAddressFromEnv(vaultClient)
	setTokenFromEnv(vaultClient)

}

func setTokenFromEnv(client *vault.Client) {
	token := os.Getenv("VAULT_TOKEN")
	if token == "" {
		log.Fatal("Unable to authenticate. No token provided.\nPlease set VAULT_TOKEN")
	}
	client.SetToken(token)
}
func setAddressFromEnv(client *vault.Client) {
	addr := os.Getenv("VAULT_ADDR")
	if addr == "" {
		log.Fatal("Unable to connect. No address provided.\nPlease set VAULT_ADDR")
	}
	client.SetAddress(addr)
}

func getEntityList(client *vault.Client) []string {

}
func getGroupList(client *vault.Client) []string {

}
func getAliasList() {

}
