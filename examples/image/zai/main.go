// Example: Z.ai Image Generation
//
// This example demonstrates how to generate images using the Iris SDK
// with Z.ai's GLM image generation models.
//
// Run with:
//
//	export ZAI_API_KEY=your-key
//	go run main.go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/erikhoward/iris/core"
	"github.com/erikhoward/iris/providers/zai"
)

func main() {
	apiKey := os.Getenv("ZAI_API_KEY")
	if apiKey == "" {
		fmt.Println("ZAI_API_KEY environment variable is required")
		os.Exit(1)
	}

	p := zai.New(apiKey)
	client := core.NewClient(p)

	fmt.Println("Generating image with Z.ai...")

	// Type assert to ImageGenerator
	imageGen, ok := client.Provider().(core.ImageGenerator)
	if !ok {
		fmt.Println("Provider does not support image generation")
		os.Exit(1)
	}

	resp, err := imageGen.GenerateImage(context.Background(), &core.ImageGenerateRequest{
		Model:   zai.ModelGLMImage,
		Prompt:  "A cute kitten sitting on a sunny windowsill with blue sky and white clouds in the background",
		Size:    core.ImageSize1024x1024,
		Quality: core.ImageQualityHigh,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if len(resp.Data) == 0 {
		fmt.Println("No images generated")
		os.Exit(1)
	}

	// Z.ai returns URLs, not base64 data
	fmt.Printf("Image URL: %s\n", resp.Data[0].URL)
	fmt.Println("\nNote: This URL expires after 30 days. Download and store the image if needed.")
}
