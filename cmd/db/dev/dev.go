package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/utils"
	"github.com/ellypaws/inkbunny/api"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

// http://localhost:1323
var apiURL = url.URL{
	Scheme: "http",
	Host:   "localhost:1323",
}

var auditor = db.Auditor{
	UserID:   int64(user.UserID),
	Username: user.Username,
	Role:     db.RoleAuditor,
}

var user = func() *api.Credentials {
	var auditor api.Credentials = api.Credentials{
		Sid:      os.Getenv("SID"),
		Username: os.Getenv("USERNAME"),
		UserID:   1,
	}
	id := os.Getenv("ID")
	if id != "" {
		auditorID, err := strconv.Atoi(id)
		if err == nil {
			auditor.UserID = api.IntString(auditorID)
		}
	}
	return &auditor
}()

var ticket *db.Ticket

var sqlite, _ = db.New(context.WithValue(context.Background(), "filename", "dev.sqlite"))

func main() {
	options()
}

func checkAuditor() {
	if user == nil {
		setAuditorID()
	}
}

func options() {
	if user != nil {
		fmt.Printf("Logged in\nUsername: %v ID: %v\n", user.Username, user.UserID)
	} else {
		fmt.Println("Logged out")
	}
	fmt.Println("------------------")
	fmt.Println("Options")
	fmt.Println("0. Set Auditor ID")
	fmt.Println("1. New ticket")
	fmt.Println("2. View tickets")
	fmt.Println("3. Search submissions")
	fmt.Println("4. New Auditor")
	fmt.Println("5. Record model hash")
	fmt.Println("all. Insert all")
	fmt.Println("exit. Exit")
	fmt.Println("------------------")
	if ticket != nil {
		fmt.Printf("Ticket: %v\n", ticket.ID)
		fmt.Println("------------------")
		status := "Close"
		if ticket.Closed {
			status = "Open"
		}
		fmt.Printf("+0. %s ticket\n", status)
		fmt.Println("+1. Set priority")
	}

	var option string
	fmt.Print("Enter option: ")
	fmt.Scanln(&option)

	switch option {
	case "":
		fmt.Println("Empty option")
		options()
	case "0":
		setAuditorID()
	case "1":
		newTicket()
	case "2":
		viewTickets()
	case "3":
		searchSubmissions()
	case "4":
		newAuditor()
	case "5":
		recordModelHash()
	case "all":
		insertAll()
	case "sid":
		setSID()
	case "exit":
		exit()
	case "+0":
		setTicketStatus()
	}

	options()
}

func recordModelHash() {
	var hash string
	fmt.Print("Enter model hash: ")
	fmt.Scanln(&hash)

	if hash == "" {
		fmt.Println("Empty model hash")
		recordModelHash()
	}

	var model string
	fmt.Print("Enter model: ")
	fmt.Scanln(&model)

	modelHash := db.ModelHashes{
		hash: []string{model},
	}

	err := sqlite.UpsertModel(modelHash)
	if err != nil {
		log.Printf("could not insert model hash: %v", err)
		recordModelHash()
	}
}

func searchSubmissions() {
	var prompt string
	fmt.Print("Enter search term: ")
	fmt.Scanln(&prompt)

	if prompt == "" {
		prompt = "ai_generated"
	}

	submissions, err := user.SearchSubmissions(
		api.SubmissionSearchRequest{
			Text:               prompt,
			SubmissionsPerPage: 5,
		},
	)
	if err != nil {
		log.Printf("could not search submissions: %v", err)
		options()
	}

	for i := range submissions.Submissions {
		fmt.Printf("Submission [%s]: %v\n", submissions.Submissions[i].SubmissionID, submissions.Submissions[i].Title)
	}

Prompt:
	prompt = ""
	fmt.Print("Enter submission ID: ")
	fmt.Scanln(&prompt)

	if prompt == "" {
		goto Prompt
	}

	submissionDetails, err := user.SubmissionDetails(api.SubmissionDetailsRequest{
		SubmissionIDs: prompt,
	})
	if err != nil {
		log.Printf("could not get submission details: %v", err)
		options()
	}

	submission := submissionDetails.Submissions[0]
	fmt.Printf("You entered: [%v]\n\n%s", prompt, submission.Description)

	var imageRequests []entities.TextToImageRequest
	for _, file := range submission.Files {
		if file.MimeType == "text/plain" {
			r, err := http.Get(file.FileURLFull)
			if err != nil {
				log.Printf("could not get file: %v", err)
			}
			defer r.Body.Close()

			b, err := io.ReadAll(r.Body)
			if err != nil {
				log.Printf("could not read file: %v", err)
			}

			_ = os.WriteFile(file.FileName, b, 0644)

			dataset := ParseDataset(utils.NameContent{file.FileName: b})

			for imageName, data := range dataset {
				_ = os.WriteFile(imageName, data, 0644)
				parameterHeuristics, err := utils.ParameterHeuristics(string(data))
				if err != nil {
					log.Printf("could not get heuristics: %v", err)
					break
				}
				imageRequests = append(imageRequests, parameterHeuristics)
			}
			break
		}
	}

	if imageRequests == nil {
		descriptionHeuristics, err := utils.DescriptionHeuristics(submissionDetails.Submissions[0].Description)
		if err != nil {
			log.Printf("could not get heuristics: %v", err)
		}
		imageRequests = append(imageRequests, descriptionHeuristics)
	}
	marshal, _ := json.MarshalIndent(imageRequests, "", "  ")
	fmt.Printf("Heuristics: %s\n", string(marshal))
}

// ParseDataset splits multiple prompts into separate keys
func ParseDataset(file utils.NameContent) utils.NameContent {
	var dataset = make(map[string][]byte)
	for name, input := range file {
		// Because some artists already have standardized txt files, opt to split each file separately
		autoSnep := strings.Contains(name, "_AutoSnep_")
		druge := strings.Contains(name, "_druge_")
		aiBean := strings.Contains(name, "_AIBean_")
		artieDragon := strings.Contains(name, "_artiedragon_")
		picker52578 := strings.Contains(name, "_picker52578_")
		if autoSnep || druge || aiBean || artieDragon || picker52578 {
			var inputResponse map[string]utils.InputResponse
			switch {
			case autoSnep:
				inputResponse = utils.MapParams(utils.AutoSnep, utils.WithBytes(input))
			case druge:
				inputResponse = utils.MapParams(utils.Common, utils.WithBytes(input), utils.UseDruge())
			case aiBean:
				inputResponse = utils.MapParams(utils.Common, utils.WithBytes(input), utils.UseAIBean())
			case artieDragon:
				inputResponse = utils.MapParams(utils.Common, utils.WithBytes(input), utils.UseArtie())
			case picker52578:
				inputResponse = utils.MapParams(
					utils.Common,
					utils.WithBytes(input),
					utils.WithFilename("picker52578_"),
					utils.WithKeyCondition(func(line string) bool { return strings.HasPrefix(line, "File Name") }))
			}
			if inputResponse != nil {
				for name, s := range inputResponse {
					if s.Input == "" {
						continue
					}
					dataset[name] = s.Response
				}
				continue
			}
		}
		dataset[name] = input
	}
	return dataset
}

func setTicketStatus() {
	ticket.Closed = !ticket.Closed
	_, err := sqlite.UpsertTicket(*ticket)
	if err != nil {
		log.Printf("could not update ticket: %v", err)
		options()
	}
}

func setSID() {
	var prompt string
	fmt.Print("Enter SID: ")
	fmt.Scanln(&prompt)

	if prompt == "" {
		fmt.Println("Empty SID")
		setSID()
	}

	if user == nil {
		user = new(api.Credentials)
	}

	user.Sid = prompt
}

func viewTickets() {
	tickets, err := sqlite.GetTicketsByAuditor(int64(user.UserID))
	if err != nil {
		log.Printf("could not get tickets: %v", err)
		options()
	}

	var validTickets []int64
	for i := range tickets {
		fmt.Printf("Ticket [%d]: %v\n", tickets[i].ID, tickets[i].Subject)
		validTickets = append(validTickets, tickets[i].ID)
	}

	var prompt string
	fmt.Print("Enter ticket ID: ")
	fmt.Scanln(&prompt)

	if prompt == "" {
		prompt = "1"
	}

	ticketID, err := strconv.ParseInt(prompt, 10, 64)
	if err != nil {
		log.Printf("Invalid ticket ID: %v", err)
		viewTickets()
	}

	if !slices.Contains(validTickets, ticketID) {
		log.Printf("Invalid ticket ID: %v", err)
		viewTickets()
	} else {
		for i := range tickets {
			if tickets[i].ID == ticketID {
				ticket = &tickets[i]
				break
			}
		}
	}

	marshal, _ := json.MarshalIndent(ticket, "", "  ")

	fmt.Printf("You entered: [%v]\n\n%s", ticketID, string(marshal))
}

func setAuditorID() {
	var prompt string
	fmt.Print("Enter auditor ID: ")
	fmt.Scanln(&prompt)

	if prompt == "" {
		setAuditorID()
	}

	auditorID, err := strconv.ParseInt(prompt, 10, 64)
	if err != nil {
		log.Printf("Invalid auditor ID: %v", err)
		setAuditorID()
	}

	fmt.Printf("You entered: [%v]\n", auditorID)

	auditor, err := sqlite.GetAuditorByID(auditorID)
	if err != nil {
		log.Printf("Invalid auditor ID: %v", err)
		setAuditorID()
	}

	user = &api.Credentials{
		UserID:   api.IntString(auditorID),
		Username: auditor.Username,
	}
}

func insertAll() {
	submissionDetails, err := user.SubmissionDetails(api.SubmissionDetailsRequest{
		SubmissionIDs: os.Getenv("SUBMISSION_ID"),
	})
	if err != nil {
		log.Fatalf("could not get submission details: %v", err)
	}

	if len(submissionDetails.Submissions) == 0 {
		log.Fatalf("no submissions found")
	}

	submission := submissionDetails.Submissions[0]
	id, _ := strconv.ParseInt(submission.SubmissionID, 10, 64)
	userID, _ := strconv.ParseInt(submission.UserID, 10, 64)

	submissionDB := db.Submission{
		ID:          id,
		UserID:      userID,
		URL:         fmt.Sprintf("https://inkbunny.net/s/%v", id),
		Title:       submission.Title,
		Description: submission.Description,
		Updated:     time.Now(),
		Metadata: db.Metadata{
			Generated: true,
		},
		Ratings: submission.Ratings,
	}

	err = sqlite.InsertSubmission(submissionDB)
	if err != nil {
		log.Fatalf("InsertSubmission() failed: %v", err)
	}

	_ = sqlite.InsertAuditor(auditor)

	audit := &db.Audit{
		AuditorID:          &auditor.UserID,
		SubmissionID:       456,
		SubmissionUsername: "User",
		SubmissionUserID:   789,
		Flags:              []db.Flag{db.FlagUndisclosed},
		ActionTaken:        "none",
	}

	_, _ = sqlite.InsertAudit(*audit)

}

func newTicket() {
	var err error
	if user == nil || user.Sid == "" {
		user, err = api.Guest().Login()
		if err != nil {
			log.Fatalf("could not login as guest: %v", err)
		}
	}

	submissions, err := user.SearchSubmissions(api.SubmissionSearchRequest{
		Text:               "ai_generated",
		SubmissionsPerPage: 1,
	})
	if err != nil || len(submissions.Submissions) == 0 {
		log.Fatalf("could not search submissions: %v", err)
	}

	submissionDetails, err := user.SubmissionDetails(api.SubmissionDetailsRequest{
		SubmissionIDs: submissions.Submissions[0].SubmissionID,
	})
	if err != nil {
		log.Fatalf("could not get submission details: %v", err)
	}

	if len(submissionDetails.Submissions) == 0 {
		log.Fatalf("no submissions found")
	}

	var submissionsIDs []int64
	var ticketLabels []db.TicketLabel
	for i := range submissionDetails.Submissions {
		submission := db.InkbunnySubmissionToDBSubmission(submissionDetails.Submissions[i])
		err := sqlite.InsertSubmission(submission)
		if err != nil {
			log.Fatalf("could not insert submission: %v", err)
		}
		id, _ := strconv.ParseInt(submissionDetails.Submissions[i].SubmissionID, 10, 64)
		submissionsIDs = append(submissionsIDs, id)
		ticketLabels = append(ticketLabels, db.SubmissionLabels(submission)...)
	}

	if len(submissionsIDs) == 0 {
		log.Fatalf("no submissions found")
	}

	err = sqlite.InsertAuditor(auditor)
	if err != nil {
		log.Fatalf("could not insert auditor: %v", err)
	}

	ticket := db.Ticket{
		ID:         1,
		Subject:    "subject",
		DateOpened: time.Now().UTC(),
		Status:     "triage",
		Labels:     ticketLabels,
		Priority:   "low",
		Closed:     false,
		Responses: []db.Response{
			{
				SupportTeam: false,
				Date:        time.Now().UTC(),
				Message:     fmt.Sprintf("The following submission doesn't include the prompts: https://inkbunny.net/s/%v", submissionsIDs[0]),
			},
		},
		SubmissionIDs: submissionsIDs,
		AssignedID:    &auditor.UserID,
		UsersInvolved: db.Involved{
			Reporter:    api.UsernameID{UserID: user.UserID.String(), Username: user.Username},
			ReportedIDs: []api.UsernameID{{UserID: "1139764", Username: "Liondaddy669"}},
		},
	}

	_, err = sqlite.UpsertTicket(ticket)
	if err != nil {
		log.Fatalf("could not insert ticket: %v", err)
	}

	fmt.Printf("Ticket created: %v\n", ticket.ID)
}

func newAuditor() {
	err := sqlite.InsertAuditor(auditor)
	if err != nil {
		log.Printf("error: could not insert auditor: %v", err)
	}
}

func exit() {
	user = nil
	main()
}
