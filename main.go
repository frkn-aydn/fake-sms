package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/anaskhan96/soup"
	"github.com/manifoldco/promptui"
)

//Number A struct that represents a new number to be addeded
type Number struct {
	Country   string `json:"country"`
	Number    string `json:"number"`
	CreatedAt string `json:"created_at"`
}

//Message a struct which represents the message
type Message struct {
	Body       string `json:"body"`
	CreatedAt  string `json:"created_at"`
	Originator string `json:"originator"`
}

//Numbers A list of Number type
type Numbers []Number

//Messages A list of Message type
type Messages []Message

func exitFatal(err error) {
	log.Fatal(err)
}

//DB The database functions group
type DB struct {
}

func (d *DB) getDBPath() string {
	/*
		Look for the path to be specified in ENV FAKE_SMS_DB_DIR,
		if not, use default $HOME as the path to create DB.
		The DB will be created at <db_dir>/.fake-sms/db.json
		If the DB does not exist, it will be created and will be
		initialized to an empty array []
	*/

	dbPath, exists := os.LookupEnv("FAKE_SMS_DB_DIR")
	if !exists {
		dbPath = os.Getenv("HOME")
		dbPath = filepath.Join(dbPath, ".fake-sms")
	}

	_, err := os.Stat(dbPath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dbPath, 0700)
		if err != nil {
			log.Fatalf("Failed to create DB directory at %s\n", dbPath)
		}
	}

	dbPath = filepath.Join(dbPath, "db.json")
	_, err = os.Stat(dbPath)
	if os.IsNotExist(err) {
		emptyArray := []byte("[\n]\n")
		err = ioutil.WriteFile(dbPath, emptyArray, 0700)
		if err != nil {
			log.Fatalf("Faild to create DB file at %s\n", dbPath)
		}
	}

	return dbPath
}

func (d *DB) addToDB(number *Number) {
	dbPath := d.getDBPath()
	//read and serialize it to numbers
	data, err := ioutil.ReadFile(dbPath)
	if err != nil {
		log.Fatalf("Failed to read DB file at %s\n", dbPath)
	}

	//unmarshall the db to Numbers type
	numbers := Numbers{}
	err = json.Unmarshal(data, &numbers)
	if err != nil {
		log.Fatalf("Failed to de-serialize DB file %s\n", dbPath)
	}

	numbers = append(numbers, *number)

	//write it back to the db
	data, err = json.Marshal(numbers)
	if err != nil {
		log.Fatalf("Failed to serialize DB file %s\n", dbPath)
	}

	err = ioutil.WriteFile(dbPath, data, 0700)
	if err != nil {
		log.Fatalf("Failed to save DB file %s\n", dbPath)
	}
}

func (d *DB) getFromDB() *Numbers {
	dbPath := d.getDBPath()
	//read and serialize it to numbers
	data, err := ioutil.ReadFile(dbPath)
	if err != nil {
		log.Fatalf("Failed to read DB file at %s\n", dbPath)
	}

	//unmarshall the db to Numbers type
	numbers := Numbers{}
	err = json.Unmarshal(data, &numbers)
	if err != nil {
		log.Fatalf("Failed to de-serialize DB file %s\n", dbPath)
	}

	return &numbers
}

func (d *DB) deleteFromDB(idx *int) {
	dbPath := d.getDBPath()
	//read and serialize it to numbers
	data, err := ioutil.ReadFile(dbPath)
	if err != nil {
		log.Fatalf("Failed to read DB file at %s\n", dbPath)
	}

	//unmarshall the db to Numbers type
	numbers := Numbers{}
	err = json.Unmarshal(data, &numbers)
	if err != nil {
		log.Fatalf("Failed to de-serialize DB file %s\n", dbPath)
	}

	//delete by index
	if *idx > len(numbers)-1 {
		log.Fatalln("Number does not exist to be deleted from DB")
	}

	numbers = append(numbers[:*idx], numbers[*idx+1:]...)
	//serialize it back
	data, err = json.Marshal(numbers)
	if err != nil {
		log.Fatalf("Failed to serialize DB file %s\n", dbPath)
	}

	err = ioutil.WriteFile(dbPath, data, 0700)
	if err != nil {
		log.Fatalf("Failed to save DB file %s\n", dbPath)
	}
}

func numbersToList(numbers *Numbers) *[]string {
	listOfNumbers := make([]string, len(*numbers))
	for idx, number := range *numbers {
		listOfNumbers[idx] = fmt.Sprintf("%s (%s)", number.Number, number.Country)
	}
	return &listOfNumbers
}

func displayInitParameters() int {
	prompt := promptui.Select{
		Label: "What you want to do?",
		Items: []string{"Add a new number", "List my numbers", "Remove a number", "Get my messages", "Exit"},
	}

	idx, _, err := prompt.Run()
	if err != nil {
		exitFatal(err)
	}

	//Return the index of the parameter selected
	return idx
}

func getAvailNumbers() *Numbers {

	numArray := ScrapeAvailableNumbers()
	numbers := Numbers(numArray)
	return &numbers
}

func registerNumber() {
	numbers := getAvailNumbers()

	if len(*numbers) == 0 {
		fmt.Println("No new numbers available right now")
	} else {
		numberList := numbersToList(numbers)
		//display numbers
		prompt := promptui.Select{
			Label: "These are the available numbers, choose any one of them",
			Items: *numberList,
		}

		idx, _, err := prompt.Run()
		if err != nil {
			exitFatal(err)
		}

		if idx == -1 {
			fmt.Println("Nothing selected")
		} else {
			//new number selected, save it to the database file
			selectedNumber := &(*numbers)[idx]
			fmt.Printf("Selected %s, saving to database\n", selectedNumber)
			db := DB{}
			db.addToDB(selectedNumber)
		}
	}
}

func listNumbers() {
	db := DB{}
	numbers := db.getFromDB()

	fmt.Println("Country\t\tNumber\t\tCreated At")
	fmt.Println("=======================================================================")
	for _, number := range *numbers {
		fmt.Printf(
			"%s\t\t%s\t\t%s\n",
			number.Country, number.Number, number.CreatedAt,
		)
	}
}

func removeNumbers() {
	db := DB{}
	numbers := db.getFromDB()

	numberList := numbersToList(numbers)

	if len(*numberList) == 0 {
		log.Fatalln("No numbers saved to delete")
	}

	//display the list
	prompt := promptui.Select{
		Label: "These are the available numbers, choose any one of them",
		Items: *numberList,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		exitFatal(err)
	}

	if idx == -1 {
		fmt.Println("Nothing selected")
	} else {
		//new number selected, save it to the database file
		selectedNumber := &(*numbers)[idx]
		fmt.Printf("Selected %s, removing from database\n", selectedNumber)
		db.deleteFromDB(&idx)
	}
}

func messagePatternCheck(pattern *string, messages *Messages) Messages {
	r, err := regexp.Compile(*pattern)
	if err != nil {
		log.Fatalln("Invalid regular expression provided")
	}

	filteredMessages := make([]Message, 0)
	for _, message := range *messages {
		//check match
		isMatch := r.Match([]byte(message.Body))
		if isMatch {
			filteredMessages = append(filteredMessages, message)
		}
	}

	return Messages(filteredMessages)
}

func checkMessages(enableFilter bool) {

	db := DB{}
	numbers := db.getFromDB()

	numberList := numbersToList(numbers)

	if len(*numberList) == 0 {
		log.Fatalln("No numbers saved to delete")
	}

	//display the list
	prompt := promptui.Select{
		Label: "These are the available numbers, choose any one of them",
		Items: *numberList,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		exitFatal(err)
	}

	if idx == -1 {
		fmt.Println("Nothing selected")
	} else {
		//new number selected, save it to the database file
		selectedNumber := &(*numbers)[idx]
		fmt.Printf("Selected %s, fetching messages\n", selectedNumber)

		messagesArray := ScrapeMessagesForNumber(selectedNumber.Number)

		//check message
		messages := Messages(messagesArray)

		//run filter if enabled:
		if enableFilter {
			fmt.Println("Enter the filter regular expression:")
			userFilterInput := ""

			fmt.Scanln(&userFilterInput)
			if userFilterInput == "" {
				userFilterInput = `.*`
			}

			//run the filter
			messages = messagePatternCheck(&userFilterInput, &messages)
		}

		fmt.Println("===========================================")
		for _, message := range messages {
			fmt.Printf("Sender : %s, at : %s\n", message.Originator, message.CreatedAt)
			fmt.Printf("Body : %s\n", message.Body)
			fmt.Println("===========================================")
		}

		indentedData, _ := json.MarshalIndent(messages, "", "\t")

		//save the body as json
		fileName := fmt.Sprintf("%s.json", selectedNumber.Number)
		err = ioutil.WriteFile(fileName, indentedData, 0700)
		if err != nil {
			log.Fatalf("Failed to save file %s\n", fileName)
		}
	}
}

func shouldIncludeFilter() bool {
	prompt := promptui.Select{
		Label: "Do you want to filter the messages?",
		Items: []string{"Yes", "No"},
	}

	idx, _, err := prompt.Run()
	if err != nil {
		log.Fatalln("Failed to render prompt")
	}

	if idx == 0 {
		return true
	}

	return false
}

func main() {

	ScrapeAvailableNumbers()

	for true {
		idx := displayInitParameters()

		switch idx {
		case 0:
			registerNumber()
			break
		case 1:
			listNumbers()
			break
		case 2:
			removeNumbers()
			break
		case 3:
			//check if filter needs to be enabled
			includeFilter := shouldIncludeFilter()
			checkMessages(includeFilter)
			break
		case 4:
			fmt.Println("Bye!")
			os.Exit(0)
		default:
			log.Fatalf("Option %d yet to be implemented\n", idx)
		}
	}
}

const (
	pageURL     = "https://receive-smss.com/"
	cookieName  = "__cfduid"
	smsEndpoint = "sms/"
)

//ScrapeAvailableNumbers Extracts the list of phone-numbers from the page
func ScrapeAvailableNumbers() []Number {
	response, err := soup.Get(pageURL)
	if err != nil {
		log.Fatalf("Failed to make HTTP request to %s\n", pageURL)
	}

	numbers := make([]Number, 0)

	//scrape the page
	document := soup.HTMLParse(response)
	numberBoxes := document.Find("div", "class", "number-boxes").FindAllStrict(
		"div", "class", "number-boxes-item d-flex flex-column ",
	)

	for _, numberBox := range numberBoxes {
		numberElement := numberBox.FindStrict("div", "class", "row")
		if numberElement.Error == nil {
			numberContainer := numberElement.FindStrict("h4")
			countryContainer := numberElement.FindStrict("h5")
			if numberContainer.Error == nil && countryContainer.Error == nil {
				number := Number{
					CreatedAt: time.Now().Format("2006-01-02 15:04:05 Monday"),
					Number:    numberContainer.Text(),
					Country:   countryContainer.Text(),
				}

				numbers = append(numbers, number)
			}
		}
	}

	return numbers
}

//ScrapeMessagesForNumber GET SMS from number
func ScrapeMessagesForNumber(number string) []Message {
	//Get cookie first
	resp, err := http.Get(pageURL)
	if err != nil {
		log.Fatalln("Failed to make GET request")
	}

	cookies := resp.Cookies()
	cookieValue := ""
	for _, cookie := range cookies {
		if cookie.Name == cookieName {
			cookieValue = cookie.Value
		}
	}

	//now use that value to set the cookie in soup
	soup.Cookie(cookieName, cookieValue)
	requestURL := pageURL + smsEndpoint + strings.ReplaceAll(number, "+", "") + "/"

	//make GET with soup:
	response, err := soup.Get(requestURL)
	if err != nil {
		log.Fatalf("error fetching data: %s", err.Error())
	}

	document := soup.HTMLParse(response)

	table := document.Find("table")
	if table.Error != nil {
		log.Fatalln("Failed to load messages")
	}

	tbody := table.Find("tbody")
	if tbody.Error != nil {
		log.Fatalln("Failed to load messages")
	}

	tableRows := tbody.FindAll("tr")

	messages := make([]Message, 0)

	for _, row := range tableRows {
		cols := row.FindAll("td")

		if len(cols) < 3 {
			continue
		}

		message := Message{
			Originator: cols[0].FullText(),
			Body:       cols[1].FullText(),
			CreatedAt:  cols[2].FullText(),
		}

		messages = append(messages, message)
	}

	return messages
}
