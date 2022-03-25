package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	vault "github.com/hashicorp/vault/api"
	profile "github.com/pkg/profile"
	progressbar "github.com/schollz/progressbar/v3"
)

func main() {
	defer profile.Start(profile.ProfilePath(".")).Stop()
	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = getAddressFromEnv()
	vaultClient, err := vault.NewClient(vaultConfig)
	if err != nil {
		log.Fatalf("Unable to initialize Vault client: %v", err)
	}

	entityIdList := getEntityIdList(vaultClient)
	groupIdList := getGroupIdList(vaultClient)
	entityDescriptions := make(chan []byte)
	groupDescriptions := make(chan []byte)

	descriptionGetterWaitGroup := sync.WaitGroup{}
	descriptionGetterWaitGroup.Add(len(entityIdList))
	descriptionGetterWaitGroup.Add(len(groupIdList))

	for _, entityId := range entityIdList {
		go getEntityDesc(vaultClient, entityId, entityDescriptions, &descriptionGetterWaitGroup)
	}
	for _, groupId := range groupIdList {
		go getGroupDesc(vaultClient, groupId, groupDescriptions, &descriptionGetterWaitGroup)
	}
	go func() {
		descriptionGetterWaitGroup.Wait()
		close(entityDescriptions)
		close(groupDescriptions)
	}()

	fmt.Println("Collecting entities and groups...")
	// Progress Bar
	progressMax := len(entityIdList) + len(groupDescriptions)
	progressChannel := make(chan bool, progressMax)
	go func() {
		bar := progressbar.Default(int64(progressMax))
		for range progressChannel {
			bar.Add(1)
		}
	}()

	fileWriterWaitGroup := sync.WaitGroup{}
	fileWriterWaitGroup.Add(2)
	go writeJsonChannelToFile(entityDescriptions, "entities.json", &fileWriterWaitGroup, progressChannel)
	go writeJsonChannelToFile(groupDescriptions, "groups.json", &fileWriterWaitGroup, progressChannel)
	fileWriterWaitGroup.Wait()
}

func writeJsonChannelToFile(channel chan []byte, fileName string, wg *sync.WaitGroup, progressChannel chan bool) {
	defer wg.Done()
	strBuilder := strings.Builder{}
	strBuilder.WriteString("{ \"list\": [\n")
	for nextBytes := range channel {
		for _, nyble := range nextBytes {
			strBuilder.WriteByte(nyble)
		}
		strBuilder.WriteString(",")
		progressChannel <- true
	}
	fmt.Printf("Writing data to %v\n", fileName)
	str := strBuilder.String()
	str = str[:len(str)-1]
	os.WriteFile(fileName, []byte(str+"\n]}"), 0644)
}

func getTokenFromEnv() string {
	token := os.Getenv("VAULT_TOKEN")
	if token == "" {
		log.Fatal("Unable to authenticate. No token provided.\nPlease set VAULT_TOKEN")
	}
	return token
}
func getAddressFromEnv() string {
	addr := os.Getenv("VAULT_ADDR")
	if addr == "" {
		log.Fatal("Unable to connect. No address provided.\nPlease set VAULT_ADDR")
	}
	return addr
}

func getEntityIdList(client *vault.Client) []string {
	listResp, err := client.Logical().List("identity/entity/id")
	if err != nil {
		log.Fatalf("unable to retrieve list of entities: %v", err)
	}

	entityIdMap := listResp.Data["key_info"].(map[string]interface{})
	entityIdList := make([]string, len(entityIdMap))
	i := 0
	for id := range entityIdMap {
		entityIdList[i] = id
		i++
	}
	return entityIdList
}
func getGroupIdList(client *vault.Client) []string {
	listResp, err := client.Logical().List("identity/group/id")
	if err != nil {
		log.Fatalf("unable to retrieve list of entities: %v", err)
	}
	groupIdMap := listResp.Data["key_info"].(map[string]interface{})
	groupIdList := make([]string, len(groupIdMap))
	i := 0
	for id := range groupIdMap {
		groupIdList[i] = id
		i++
	}
	return groupIdList
}
func getEntityDesc(vaultClient *vault.Client, entityId string, entityDesc chan []byte, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	readResp, err := vaultClient.Logical().Read("identity/entity/id/" + entityId)
	if err != nil {
		fmt.Printf("WARNING: Could not get entity with ID: %v: %v\n", entityId, err)
		return
	}
	jsonBytes, err := json.MarshalIndent(readResp.Data, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	entityDesc <- jsonBytes
}

func getGroupDesc(vaultClient *vault.Client, groupId string, groupDescriptions chan []byte, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	readResp, err := vaultClient.Logical().Read("identity/group/id/" + groupId)
	if err != nil {
		fmt.Printf("WARNING: Could not get group with ID: %v: %v\n", groupId, err)
		return
	}
	jsonBytes, err := json.MarshalIndent(readResp.Data, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	groupDescriptions <- jsonBytes
}
