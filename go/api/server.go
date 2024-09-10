package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/anmho/prism/scope"

	"github.com/google/generative-ai-go/genai"
	"github.com/sashabaranov/go-openai"
	"google.golang.org/api/iterator"
)

func MakeServer(openaiClient *openai.Client, googleClient *genai.Client) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("string"))
	})
	mux.HandleFunc("GET /api/chat", handleChatCompletions(openaiClient, googleClient))
	mux.Handle("GET /", http.FileServer(http.Dir("./chat/dist")))
	mux.HandleFunc("GET /api/events", eventsHandler)

	return mux
}

type Model int

const (
	UnsetModel Model = iota
	Gemini1Dot5Flash
	GPT3Dot5
)

func (m Model) String() string {
	switch m {
	case Gemini1Dot5Flash:
		return "gemini-1.5-flash"
	case GPT3Dot5:
		return "gpt3.5"
	default:
		return "unknown"
	}
}

type ChatParams struct {
	Model Model
}

type StreamableLLM struct {
}

func modelFromString(modelName string) Model {
	switch modelName {
	case Gemini1Dot5Flash.String():
		return Gemini1Dot5Flash
	case GPT3Dot5.String():
		return GPT3Dot5
	default:
		return UnsetModel
	}
}

type PromptInstructions struct {
	Role   string `json:"role"`
	Prompt string `json:"prompt"`
}

type PromptData struct {
	Prompt       string             `json:"prompt"`
	Instructions PromptInstructions `json:"instructions"`
}

type ResponseBlock struct {
	Text string `json:"text"`
}

func handleChatCompletions(openaiClient *openai.Client, googleClient *genai.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "text/event-stream")
		//
		//prompt := `
		//You're planning a magical trip to Disneyland Anaheim for a family of four, including two adults and two children aged 7 and 10.
		//The family is visiting for three days and wants to make the most of their experience, including exploring all the major attractions,
		//enjoying Disney-themed dining, and making sure they don't miss any must-see shows or parades.
		//
		//They prefer a balanced mix of thrill rides and kid-friendly attractions, and they want to minimize wait times as much as possible.
		//The family is staying at one of the Disneyland Resort hotels, and they'd like to take advantage of early park entry and any other benefits that come with their stay.
		//Please create a detailed three-day itinerary, including recommendations for attractions, dining reservations, and tips on the best times to visit each area of the park.
		//Be sure to include advice on how to navigate the park efficiently and any insider tips that would make the trip extra special.
		//Feel free to suggest specific themes for each day (e.g., a "Star Wars" day in Galaxy's Edge), and include any seasonal events or limited-time experiences happening during their visit.
		//`

		var promptData PromptData
		err := json.Unmarshal([]byte(r.URL.Query().Get("data")), &promptData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		slog.Info("parsing llm prompt data", slog.Any("promptData", promptData))

		modelName := r.URL.Query().Get("model")
		modelType := modelFromString(modelName)

		slog.Info("handling chat completion", slog.String("model", modelName))

		ctx := r.Context()

		switch modelType {
		case Gemini1Dot5Flash:
			model := googleClient.GenerativeModel(Gemini1Dot5Flash.String())

			iter := model.GenerateContentStream(ctx, genai.Text(
				fmt.Sprintf(
					`
				%s: %s
				User: %s
				`,
					promptData.Instructions.Role, promptData.Instructions.Prompt,
					promptData.Prompt,
				),
			))
			for {
				resp, err := iter.Next()
				if err != nil {
					if errors.Is(err, iterator.Done) {
						fmt.Println("\nStream finished")
						break
					}

					if err != nil {
						fmt.Printf("\nStream error: %s\n", err.Error())
						break
					}
				}

				fmt.Println("Number of response candidates: ", len(resp.Candidates))
				for _, part := range resp.Candidates[0].Content.Parts {

					switch p := part.(type) {
					case genai.Text:
						block := ResponseBlock{
							Text: string(p),
						}

						b, err := json.Marshal(block)
						if err != nil {
							scope.GetLogger().Error("serializing text", slog.Any("err", err))
							continue
						}
						fmt.Fprintf(w, "data: %s\n\n", string(b))
					}

					w.(http.Flusher).Flush()
				}
			}

		case GPT3Dot5:
			req := openai.ChatCompletionRequest{
				Model: openai.GPT3Dot5Turbo,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    "system",
						Content: promptData.Instructions.Prompt,
					},
				},
				Stream: true,
			}
			stream, err := openaiClient.CreateChatCompletionStream(ctx, req)
			if err != nil {
				log.Println("error", err)
				return
			}
			defer stream.Close()

			for {
				response, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					fmt.Println("\nStream finished")
					break
				}

				if err != nil {
					fmt.Printf("\nStream error: %v\n", err)
					break
				}

				token := response.Choices[0].Delta.Content
				fmt.Fprintf(w, "data: %s\n\n", token)

				w.(http.Flusher).Flush()
			}

		default:
			http.NotFound(w, r)
		}

		w.WriteHeader(http.StatusOK)
		// doneChan := r.Context().Done()
		// <-doneChan
	}

}

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "text/event-stream")

	text := `
	As the sun dipped below the horizon, the city skyline transformed into a sea of glowing lights, each window a story, each street a vein pulsing with life. The air was thick with the scent of rain-soaked pavement, mingling with the aroma of street food wafting from hidden alleys. In the distance, the hum of distant conversations blended with the rhythmic beat of a street musician’s guitar, creating a symphony that only the night could compose.

	Among the crowd, a lone figure moved with purpose, their footsteps echoing softly against the cobblestones. Clad in a worn leather jacket, their eyes scanned the bustling streets, searching for something—or perhaps someone. The city's energy coursed through them, a silent companion on their nocturnal quest. As they turned a corner, a flash of neon light reflected in their eyes, revealing a hidden smile.

	This was a place where secrets whispered through the cracks in the pavement, where dreams danced in the shadows, waiting to be claimed by those brave enough to chase them. Here, in the heart of the night, anything was possible.
	`

	words := strings.Split(text, " ")
	for i, word := range words {

		fmt.Fprintf(w, "data: %s\n\n", word)
		if i%15 == 0 {
			fmt.Fprintf(w, "\n\n\n")
		}
		time.Sleep(50 * time.Millisecond)
		w.(http.Flusher).Flush()
	}

	doneChan := r.Context().Done()
	<-doneChan

}
