package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type StreamingInput struct {
	Content   string `json:"content"`
	ChunkSize int    `json:"chunk_size"`
}

type StreamingOutput struct {
	Message     string `json:"message"`
	TotalChunks int    `json:"total_chunks"`
}

const sampleText = `
The Go programming language is an open source project to make programmers more productive.
Go is expressive, concise, clean, and efficient. Its concurrency mechanisms make it easy to
write programs that get the most out of multicore and networked machines, while its novel
type system enables flexible and modular program construction. Go compiles quickly to
machine code yet has the convenience of garbage collection and the power of run-time reflection.
It's a fast, statically typed, compiled language that feels like a dynamically typed, interpreted language.
`

func main() {
	// Create a new Hatchet client
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// Create a workflow for streaming
	workflow := client.NewWorkflow("streaming-workflow")

	// Define the streaming task
	workflow.NewTask("stream-content", func(ctx hatchet.Context, input StreamingInput) (StreamingOutput, error) {
		content := input.Content
		if content == "" {
			content = sampleText
		}

		chunkSize := input.ChunkSize
		if chunkSize <= 0 {
			chunkSize = 50
		}

		// Split content into chunks and stream them
		chunks := createChunks(content, chunkSize)

		log.Printf("Starting to stream %d chunks...", len(chunks))

		for i, chunk := range chunks {
			// Stream each chunk
			ctx.PutStream(fmt.Sprintf("Chunk %d: %s", i+1, strings.TrimSpace(chunk)))

			// Small delay between chunks to simulate processing
			time.Sleep(300 * time.Millisecond)
		}

		ctx.PutStream("Streaming completed!")

		return StreamingOutput{
			Message:     "Content streaming finished",
			TotalChunks: len(chunks),
		}, nil
	})

	// Create a worker to run the workflow
	worker, err := client.NewWorker("streaming-worker", hatchet.WithWorkflows(workflow))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	// Start the worker in a goroutine
	go func() {
		log.Println("Starting streaming worker...")
		if err := worker.StartBlocking(); err != nil {
			log.Printf("worker failed: %v", err)
		}
	}()

	// Wait a moment for the worker to start
	time.Sleep(2 * time.Second)

	// Start HTTP server to demonstrate streaming
	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		// Set headers for streaming response
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Run the streaming workflow
		workflowRun, err := client.RunNoWait(ctx, "streaming-workflow", StreamingInput{
			Content:   sampleText,
			ChunkSize: 80,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to run workflow: %v", err), http.StatusInternalServerError)
			return
		}

		// Subscribe to the stream
		stream, err := client.Runs().SubscribeToStream(ctx, workflowRun.RunId)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to subscribe to stream: %v", err), http.StatusInternalServerError)
			return
		}

		// Stream the content to the HTTP response
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		for message := range stream {
			fmt.Fprintf(w, "data: %s\n\n", message)
			flusher.Flush()
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>Hatchet Streaming Example</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 800px; margin: 0 auto; }
        .output { 
            border: 1px solid #ccc; 
            padding: 20px; 
            height: 400px; 
            overflow-y: auto; 
            background-color: #f5f5f5;
            white-space: pre-wrap;
        }
        button { 
            padding: 10px 20px; 
            font-size: 16px; 
            margin: 10px 0; 
            background-color: #007cba; 
            color: white; 
            border: none; 
            cursor: pointer; 
        }
        button:hover { background-color: #005a87; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Hatchet Streaming Example</h1>
        <p>Click the button below to start streaming content from a Hatchet workflow:</p>
        <button onclick="startStream()">Start Streaming</button>
        <button onclick="clearOutput()">Clear Output</button>
        <div id="output" class="output"></div>
    </div>

    <script>
        function startStream() {
            const output = document.getElementById('output');
            output.innerHTML = 'Starting stream...\n';
            
            fetch('/stream')
                .then(response => {
                    const reader = response.body.getReader();
                    const decoder = new TextDecoder();
                    
                    function readStream() {
                        reader.read().then(({ done, value }) => {
                            if (done) {
                                output.innerHTML += '\nStream completed.\n';
                                return;
                            }
                            
                            const chunk = decoder.decode(value);
                            const lines = chunk.split('\n');
                            
                            lines.forEach(line => {
                                if (line.startsWith('data: ')) {
                                    output.innerHTML += line.substring(6) + '\n';
                                    output.scrollTop = output.scrollHeight;
                                }
                            });
                            
                            readStream();
                        });
                    }
                    
                    readStream();
                })
                .catch(err => {
                    output.innerHTML += 'Error: ' + err.message + '\n';
                });
        }
        
        function clearOutput() {
            document.getElementById('output').innerHTML = '';
        }
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	})

	log.Println("Starting HTTP server on :8080...")
	log.Println("Visit http://localhost:8080 to see the streaming example")
	log.Fatal(http.ListenAndServe(":8080", nil)) //nolint:gosec // This is a demo
}

func createChunks(content string, chunkSize int) []string {
	var chunks []string
	words := strings.Fields(strings.TrimSpace(content))

	currentChunk := ""
	for _, word := range words {
		if len(currentChunk)+len(word)+1 > chunkSize && currentChunk != "" {
			chunks = append(chunks, currentChunk)
			currentChunk = word
		} else {
			if currentChunk != "" {
				currentChunk += " "
			}
			currentChunk += word
		}
	}

	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}
