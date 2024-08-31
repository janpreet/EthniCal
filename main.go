package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	ics "github.com/arran4/golang-ical"
)

type AIProvider interface {
	Query(prompt string) (string, error)
}

type ClaudeProvider struct {
	ApiKey string
	Model  string
}

type OpenAIProvider struct {
	ApiKey string
	Model  string
}

func (p *OpenAIProvider) Query(prompt string) (string, error) {
	client := openai.NewClient(p.ApiKey)
	resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: p.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "You are a helpful assistant.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 4000,
	})
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

func (p *ClaudeProvider) Query(prompt string) (string, error) {
    apiUrl := "https://api.anthropic.com/v1/messages"

    payload := map[string]interface{}{
        "model":      p.Model,
        "max_tokens": 1024,
        "messages": []map[string]string{
            {"role": "user", "content": prompt},
        },
    }
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return "", err
    }

    req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(payloadBytes))
    if err != nil {
        return "", err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("x-api-key", p.ApiKey)
    req.Header.Set("anthropic-version", "2023-06-01")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    var result map[string]interface{}
    if err := json.Unmarshal(body, &result); err != nil {
        return "", err
    }

    content, ok := result["content"].([]interface{})
    if !ok || len(content) == 0 {
        return "", fmt.Errorf("unexpected response format: %s", body)
    }

    firstContent, ok := content[0].(map[string]interface{})
    if !ok {
        return "", fmt.Errorf("unexpected content format: %v", content[0])
    }

    text, ok := firstContent["text"].(string)
    if !ok {
        return "", fmt.Errorf("unexpected text format: %v", firstContent["text"])
    }

    return text, nil
}

func getAIProvider(providerName, apiKey, model string) (AIProvider, error) {
	switch providerName {
	case "openai":
		return &OpenAIProvider{
			ApiKey: apiKey,
			Model:  model,
		}, nil
	case "claude":
		return &ClaudeProvider{
			ApiKey: apiKey,
			Model:  model,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", providerName)
	}
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
	prompt := fmt.Sprintf("Please provide a list of events for %s for the current year. Use your knowledge base and ensure cultural accuracy. %s Format each event as 'Event Name: YYYY-MM-DD'.",
		item.Name, item.AdditionalInfo)

	fmt.Printf("Querying AI for %s with prompt:\n%s\n", item.Name, prompt)

	response, err := ai.Query(prompt)
	if err != nil {
		return nil, fmt.Errorf("AI query error for %s: %v", item.Name, err)
	}

	fmt.Printf("Raw AI response for %s:\n%s\n", item.Name, response)

	events, err := parseEvents(response, item.Name)
	if err != nil {
		return nil, fmt.Errorf("Error parsing events for %s: %v", item.Name, err)
	}

	fmt.Printf("Parsed events for %s:\n%v\n", item.Name, events)

	return events, nil
}

func parseEvents(aiResponse, itemName string) ([]Event, error) {
	events := []Event{}
	lines := strings.Split(aiResponse, "\n")
	fmt.Printf("Parsing %d lines for %s\n", len(lines), itemName)
	for _, line := range lines {
		parts := strings.Split(line, ": ")
		if len(parts) != 2 {
			fmt.Printf("Skipping invalid line: %s\n", line)
			continue
		}
		date, err := time.Parse("2006-01-02", parts[1])
		if err != nil {
			fmt.Printf("Error parsing date %s: %v\n", parts[1], err)
			continue
		}
		events = append(events, Event{
			Name: parts[0],
			Date: date,
			Item: itemName,
		})
	}
	fmt.Printf("Parsed %d events for %s\n", len(events), itemName)
	return events, nil
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
	if len(events) == 0 {
		fmt.Println("No events to generate ICS files for.")
		return nil
	}

	fmt.Printf("Generating ICS files for %d events and %d group configs\n", len(events), len(groupConfigs))

	for _, groupConfig := range groupConfigs {
		groupEvents := filterEventsByGroup(events, groupConfig.GroupName)

		if len(groupEvents) > 0 {
			cal := createCalendar(groupEvents, groupConfig.GroupName)
			filename := filepath.Join("docs", fmt.Sprintf("%s_events.ics", strings.ToLower(strings.ReplaceAll(groupConfig.GroupName, " ", "_"))))
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

		for _, item := range groupConfig.CalendarItems {
			itemEvents := filterEventsByItem(events, item.Name)
			if len(itemEvents) > 0 {
				cal := createCalendar(itemEvents, item.Name)
				filename := filepath.Join("docs", fmt.Sprintf("%s_%s_events.ics", strings.ToLower(strings.ReplaceAll(groupConfig.GroupName, " ", "_")), strings.ToLower(strings.ReplaceAll(item.Name, " ", "_"))))
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
		}
	}

	if len(events) > 0 {
		allCal := createCalendar(events, "All Events")
		allFile, err := os.Create(filepath.Join("docs", "all_events.ics"))
		if err != nil {
			return err
		}
		err = allCal.SerializeTo(allFile)
		allFile.Close()
		if err != nil {
			return err
		}
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
    fmt.Println("Generating HTML calendar")

    funcMap := template.FuncMap{
        "ToLower":       strings.ToLower,
        "ReplaceSpaces": func(s string) string { return strings.ReplaceAll(s, " ", "_") },
    }

    tmpl, err := template.New("calendar").Funcs(funcMap).ParseFiles("calendar_template.html")
    if err != nil {
        return fmt.Errorf("Error parsing HTML template: %v", err)
    }

    file, err := os.Create(filepath.Join("docs", "index.html"))
    if err != nil {
        return fmt.Errorf("Error creating index.html: %v", err)
    }
    defer file.Close()

    data := struct {
        Events       []Event
        GroupConfigs []GroupConfig
    }{
        Events:       events,
        GroupConfigs: groupConfigs,
    }

    err = tmpl.ExecuteTemplate(file, "calendar_template.html", data)
    if err != nil {
        return fmt.Errorf("Error writing to index.html: %v", err)
    }

    fmt.Println("HTML calendar generated successfully")
    return nil
}

func main() {
	apiKey := os.Getenv("AI_API_KEY")
	model := os.Getenv("AI_MODEL")
	disableAI := os.Getenv("DISABLE_AI")

	groupConfigs, err := loadGroupConfigs("configs")
	if err != nil {
		fmt.Printf("Error loading group configs: %v\n", err)
		return
	}
	fmt.Printf("Loaded %d group configs\n", len(groupConfigs))

	var allEvents []Event

	for _, groupConfig := range groupConfigs {
		fmt.Printf("Processing group: %s\n", groupConfig.GroupName)

		if strings.ToLower(disableAI) == "true" {
			fmt.Println("AI queries are disabled, skipping AI calls.")
			continue
		}

		ai, err := getAIProvider(groupConfig.AIProvider, apiKey, model)
		if err != nil {
			fmt.Printf("Error getting AI provider for group %s: %v\n", groupConfig.GroupName, err)
			continue
		}

		for _, item := range groupConfig.CalendarItems {
			fmt.Printf("Querying AI for item: %s\n", item.Name)
			events, err := queryAI(item, ai)
			if err != nil {
				fmt.Printf("Error querying AI for %s events in group %s: %v\n", item.Name, groupConfig.GroupName, err)
				continue
			}

			fmt.Printf("Generated %d events for %s\n", len(events), item.Name)

			for i := range events {
				events[i].Group = groupConfig.GroupName
			}

			allEvents = append(allEvents, events...)
		}
	}

	fmt.Printf("Total number of events generated: %d\n", len(allEvents))

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

	fmt.Println("Calendar files have been created successfully in the docs directory.")
}
