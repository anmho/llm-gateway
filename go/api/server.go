package api

import (
	"errors"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"github.com/sashabaranov/go-openai"
	"google.golang.org/api/iterator"
	"io"
	"log"
	"net/http"
)

func MakeServer(openaiClient *openai.Client, googleClient *genai.Client) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /chat", handleChatCompletions(openaiClient, googleClient))
	//mux.Handle("GET /", http.FileServer(http.Dir("./static")))
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
func handleChatCompletions(openaiClient *openai.Client, googleClient *genai.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "text/event-stream")

		prompt := `
		You're planning a magical trip to Disneyland Anaheim for a family of four, including two adults and two children aged 7 and 10. 
		The family is visiting for three days and wants to make the most of their experience, including exploring all the major attractions, 
		enjoying Disney-themed dining, and making sure they don't miss any must-see shows or parades. 

		They prefer a balanced mix of thrill rides and kid-friendly attractions, and they want to minimize wait times as much as possible. 
		The family is staying at one of the Disneyland Resort hotels, and they'd like to take advantage of early park entry and any other benefits that come with their stay. 
		Please create a detailed three-day itinerary, including recommendations for attractions, dining reservations, and tips on the best times to visit each area of the park. 
		Be sure to include advice on how to navigate the park efficiently and any insider tips that would make the trip extra special. 
		Feel free to suggest specific themes for each day (e.g., a "Star Wars" day in Galaxy's Edge), and include any seasonal events or limited-time experiences happening during their visit.
		`

		modelName := r.URL.Query().Get("model")
		modelType := modelFromString(modelName)

		ctx := r.Context()

		switch modelType {
		case Gemini1Dot5Flash:
			model := googleClient.GenerativeModel(Gemini1Dot5Flash.String())
			iter := model.GenerateContentStream(ctx, genai.Text(prompt))

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
				//for _, cand := range resp.Candidates {
				for _, part := range resp.Candidates[0].Content.Parts {
					fmt.Fprintf(w, "data: %s\n\n", part)

					w.(http.Flusher).Flush()
				}
				//}
			}

		case GPT3Dot5:
			req := openai.ChatCompletionRequest{
				Model: openai.GPT3Dot5Turbo,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    "system",
						Content: prompt,
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
		doneChan := r.Context().Done()
		<-doneChan

	}

}

func printResponse(resp *genai.GenerateContentResponse) {
	fmt.Println("Number of response candidates: ", len(resp.Candidates))
	for _, cand := range resp.Candidates {
		for _, part := range cand.Content.Parts {
			fmt.Println(part)
		}
	}
	fmt.Println("---")
}
