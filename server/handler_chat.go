// Copyright 2025 Google LLC
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"regexp"

	"google.golang.org/genai"
)

var model = flag.String("model", "gemini-2.0-flash", "gemini-2.0-flash")

func (cfg *apiConfig) HandlerChat(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Chat string `json:"chat"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		ErrorBadRequest("failed to parse request body", w, nil)
		return
	}
	fmt.Printf("params.Chat: %v\n", params.Chat)

	part := genai.Part{Text: params.Chat}

	result, err := cfg.chat.SendMessage(req.Context(), part)
	if err != nil {
		ErrorServer("failed to get response", w, err)
		return
	}

	text := result.Text()
	pattern := `(?is)` + `\s*(\{.*\})\s*`
	regex := regexp.MustCompile(pattern)
	matches := regex.FindStringSubmatch(text)
	fmt.Println(text)
	fmt.Println("first text")

	if len(matches) > 1 {
		// Found a JSON response, try to parse it as a tool call
		var toolCall struct {
			Tool      string         `json:"tool"`
			Arguments map[string]any `json:"arguments"`
		}
		err := json.Unmarshal([]byte(matches[1]), &toolCall)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			fmt.Printf("matches: %v\n", matches)
			return
		}
		fmt.Printf("toolCall: %v\n", toolCall)
		// Execute the tool and get the result
		toolResult := cfg.ExecuteTool(toolCall.Tool, toolCall.Arguments)

		// Send the tool result back to the LLM for processing
		followUpPrompt := fmt.Sprintf("Tool '%s' was executed with result: %s\n\nPlease provide a natural response to the user based on this result.IMPORTANT do not make further tool calls",
			toolCall.Tool, toolResult)
		fmt.Printf("followUpPrompt: %v\n", followUpPrompt)

		part = genai.Part{Text: followUpPrompt}
		fmt.Printf("%v", result)

		result, err = cfg.chat.SendMessage(req.Context(), part)
		if err != nil {
			ErrorServer("failed to process tool result", w, err)
			return
		}
		text = result.Text()
		fmt.Println(text)
	}

	RetVal := struct{ Response string }{Response: text}
	dat, err := json.Marshal(RetVal)

	if err != nil {
		ErrorServer("failed to parse chats to response", w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}
