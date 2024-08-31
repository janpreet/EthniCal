package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
	"html/template"

	ics "github.com/arran4/golang-ical"
)

type AIProvider interface {
	Query(prompt string) (string, error)
}

type OpenAIProvider struct {
	ApiKey string
}

func (p *OpenAIProvider) Query(prompt string) (string, error) {
	return fmt.Sprintf("Simulated OpenAI response for: %s", prompt), nil
}

type ClaudeProvider struct {
	ApiKey string
}

func (p *ClaudeProvider) Query(prompt string) (string, error) {
	return fmt.Sprintf("Simulated Claude AI response for: %s", prompt), nil
}

type CalendarItem struct {
	Name           string `json:"name"`
	AuthorityURL   string `json:"authorityUrl,omitempty"`
	AdditionalInfo string `json:"additionalInfo,omitempty"`
}

type GroupConfig struct {
	GroupName     string         `json:"groupName"`
	CalendarItems []CalendarItem `json:"calendarItems"`
	AIProvider    string         `json:"aiProvider"`
}

type Event struct {
	Name  string
	Date  time.Time
	Item  string
	Group string
}

func loadGroupConfigs(configDir string) ([]GroupConfig, error) {
	var groupConfigs []GroupConfig
	files, err := ioutil.ReadDir(configDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			data, err := ioutil.ReadFile(filepath.Join(configDir, file.Name()))
			if err != nil {
				return nil, err
			}

			var config GroupConfig
			err = json.Unmarshal(data, &config)
			if err != nil {
				return nil, err
			}

			groupConfigs = append(groupConfigs, config)
		}
	}

	return groupConfigs, nil
}

func queryAI(item CalendarItem, ai AIProvider) ([]Event, error) {
	var prompt string
	if item.AuthorityURL != "" {
		prompt = fmt.Sprintf("Please provide a list of events for %s for the current year. Use %s as your primary source of information. %s Format each event as 'Event Name: YYYY-MM-DD'.", 
			item.Name, item.AuthorityURL, item.AdditionalInfo)
	} else {
		prompt = fmt.Sprintf("Please provide a list of important events, holidays, and observances for %s for the current year. Use your knowledge base and ensure cultural accuracy. %s Format each event as 'Event Name: YYYY-MM-DD'.", 
			item.Name, item.AdditionalInfo)
	}

	response, err := ai.Query(prompt)
	if err != nil {
		return nil, fmt.Errorf("AI query error: %v", err)
	}

	return parseEvents(response, item.Name)
}

func parseEvents(aiResponse, itemName string) ([]Event, error) {
	events := []Event{}
	lines := strings.Split(aiResponse, "\n")
	for _, line := range lines {
		parts := strings.Split(line, ": ")
		if len(parts) != 2 {
			continue
		}
		date, err := time.Parse("2006-01-02", parts[1])
		if err != nil {
			continue
		}
		events = append(events, Event{
			Name: parts[0],
			Date: date,
			Item: itemName,
		})
	}
	return events, nil
}

func validateEvents(events []Event, item CalendarItem) ([]Event, error) {
	validatedEvents := []Event{}
	for _, event := range events {
		if event.Date.Year() == time.Now().Year() && event.Date.After(time.Now()) {
			validatedEvents = append(validatedEvents, event)
		}
	}
	return validatedEvents, nil
}

func createCalendar(events []Event, name string) *ics.Calendar {
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)
	cal.SetCalscale("GREGORIAN")
	cal.SetName(fmt.Sprintf("Global Calendar - %s", name))
	cal.SetDescription(fmt.Sprintf("AI-generated calendar of events for %s", name))

	for _, event := range events {
		icsEvent := cal.AddEvent(fmt.Sprintf("%s-%d", event.Name, event.Date.Year()))
		icsEvent.SetCreatedTime(time.Now())
		icsEvent.SetDtStampTime(time.Now())
		icsEvent.SetModifiedAt(time.Now())
		icsEvent.SetStartAt(event.Date)
		icsEvent.SetEndAt(event.Date.Add(24 * time.Hour))
		icsEvent.SetSummary(fmt.Sprintf("%s (%s)", event.Name, event.Item))
		icsEvent.SetDescription(fmt.Sprintf("%s event", event.Group))
	}

	return cal
}

func generateICSFiles(events []Event, groupConfigs []GroupConfig) error {
	for _, event := range events {
		itemEvents := filterEventsByItem(events, event.Item)
		cal := createCalendar(itemEvents, event.Item)
		
		filename := fmt.Sprintf("docs/%s_%s_events.ics", 
			strings.ToLower(strings.ReplaceAll(event.Group, " ", "_")),
			strings.ToLower(strings.ReplaceAll(event.Item, " ", "_")))
		file, err := os.Create(filename)
		if err != nil {
			return err
		}
		err = cal.SerializeTo(file)
		file.Close()
		if err != nil {
			return err
		}
	}

	for _, group := range groupConfigs {
		groupEvents := filterEventsByGroup(events, group.GroupName)
		cal := createCalendar(groupEvents, group.GroupName)
		
		filename := fmt.Sprintf("docs/%s_events.ics", 
			strings.ToLower(strings.ReplaceAll(group.GroupName, " ", "_")))
		file, err := os.Create(filename)
		if err != nil {
			return err
		}
		err = cal.SerializeTo(file)
		file.Close()
		if err != nil {
			return err
		}
	}

	allCal := createCalendar(events, "All Events")
	allFile, err := os.Create("docs/all_events.ics")
	if err != nil {
		return err
	}
	err = allCal.SerializeTo(allFile)
	allFile.Close()
	if err != nil {
		return err
	}

	return nil
}

func filterEventsByItem(events []Event, item string) []Event {
	var filtered []Event
	for _, event := range events {
		if event.Item == item {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func filterEventsByGroup(events []Event, group string) []Event {
	var filtered []Event
	for _, event := range events {
		if event.Group == group {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func generateHTMLCalendar(events []Event, groupConfigs []GroupConfig) error {
	tmpl, err := template.New("calendar").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Global Calendar</title>
    <style>
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid black; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .hidden { display: none; }
    </style>
</head>
<body>
    <h1>Global Calendar</h1>
    <form id="calendarForm">
        {{range .GroupConfigs}}
        <fieldset>
            <legend>{{.GroupName}}</legend>
            {{range .CalendarItems}}
            <label>
                <input type="checkbox" name="item" value="{{.Name}}" data-group="{{$.GroupName}}">
                {{.Name}}
            </label>
            {{end}}
        </fieldset>
        {{end}}
    </form>
    <div id="icsLinks">
        <h3>Download ICS Files:</h3>
        <ul>
            <li><a href="all_events.ics">All Events</a></li>
            {{range .GroupConfigs}}
            <li><a href="{{.GroupName | ToLower | ReplaceSpaces}}_events.ics">{{.GroupName}}</a></li>
            {{range .CalendarItems}}
            <li><a href="{{$.GroupName | ToLower | ReplaceSpaces}}_{{.Name | ToLower | ReplaceSpaces}}_events.ics">{{$.GroupName}} - {{.Name}}</a></li>
            {{end}}
            {{end}}
        </ul>
    </div>
    <table id="eventTable">
        <tr>
            <th>Date</th>
            <th>Event</th>
            <th>Item</th>
            <th>Group</th>
        </tr>
        {{range .Events}}
        <tr class="event-row" data-item="{{.Item}}" data-group="{{.Group}}">
            <td>{{.Date.Format "2006-01-02"}}</td>
            <td>{{.Name}}</td>
            <td>{{.Item}}</td>
            <td>{{.Group}}</td>
        </tr>
        {{end}}
    </table>
    <h2>Feedback</h2>
    <p>If you notice any inaccuracies, please <a href="https://github.com/janpreet/EthniCal/issues/new" target="_blank">open an issue on GitHub</a>.</p>
    <script>
    document.addEventListener('DOMContentLoaded', function() {
        const calendarForm = document.getElementById('calendarForm');
        const rows = document.getElementsByClassName('event-row');

        function updateCalendar() {
            const selectedItems = Array.from(calendarForm.querySelectorAll('input[name="item"]:checked'))
                .map(checkbox => checkbox.value);

            for (let row of rows) {
                const item = row.getAttribute('data-item');
                if (selectedItems.length === 0 || selectedItems.includes(item)) {
                    row.classList.remove('hidden');
                } else {
                    row.classList.add('hidden');
                }
            }
        }

        calendarForm.addEventListener('change', updateCalendar);
        updateCalendar();
    });
    </script>
</body>
</html>
`)
	if err != nil {
		return err
	}

	file, err := os.Create("docs/index.html")
	if err != nil {
		return err
	}
	defer file.Close()

	data := struct {
		Events       []Event
		GroupConfigs []GroupConfig
	}{
		Events:       events,
		GroupConfigs: groupConfigs,
	}

	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
		"ReplaceSpaces": func(s string) string {
			return strings.ReplaceAll(s, " ", "_")
		},
	}

	tmpl = tmpl.Funcs(funcMap)
	return tmpl.Execute(file, data)
}

func main() {
	groupConfigs, err := loadGroupConfigs("configs")
	if err != nil {
		fmt.Printf("Error loading group configs: %v\n", err)
		return
	}

	var allEvents []Event
	for _, groupConfig := range groupConfigs {
		var ai AIProvider
		switch groupConfig.AIProvider {
		case "openai":
			ai = &OpenAIProvider{ApiKey: os.Getenv("OPENAI_API_KEY")}
		case "claude":
			ai = &ClaudeProvider{ApiKey: os.Getenv("CLAUDE_API_KEY")}
		default:
			fmt.Printf("Unknown AI provider for group %s: %s\n", groupConfig.GroupName, groupConfig.AIProvider)
			continue
		}

		for _, item := range groupConfig.CalendarItems {
			events, err := queryAI(item, ai)
			if err != nil {
				fmt.Printf("Error querying AI for %s events in group %s: %v\n", item.Name, groupConfig.GroupName, err)
				continue
			}
			
			validatedEvents, err := validateEvents(events, item)
			if err != nil {
				fmt.Printf("Error validating events for %s in group %s: %v\n", item.Name, groupConfig.GroupName, err)
				continue
			}
			
			for i := range validatedEvents {
				validatedEvents[i].Group = groupConfig.GroupName
			}
			
			allEvents = append(allEvents, validatedEvents...)
		}
	}

	err = generateICSFiles(allEvents, groupConfigs)
	if err != nil {
		fmt.Printf("Error generating ICS files: %v\n", err)
		return
	}

	err = generateHTMLCalendar(allEvents, groupConfigs)
	if err != nil {
		fmt.Printf("Error generating HTML calendar: %v\n", err)
		return
	}

	fmt.Println("Calendar files have been created successfully.")
}