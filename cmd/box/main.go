package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/YouEclipse/steam-box/pkg/steambox"
	"github.com/google/go-github/github"
)

func main() {
	steamAPIKey := os.Getenv("STEAM_API_KEY")
	steamID, _ := strconv.ParseUint(os.Getenv("STEAM_ID"), 10, 64)
	appIDs := os.Getenv("APP_ID")
	appIDList := make([]uint32, 0)

	for _, appID := range strings.Split(appIDs, ",") {
		appid, err := strconv.ParseUint(appID, 10, 32)
		if err != nil {
			continue
		}
		appIDList = append(appIDList, uint32(appid))
	}

	ghToken := os.Getenv("GH_TOKEN")
	ghUsername := os.Getenv("GH_USER")
	gistID := os.Getenv("GIST_ID")				// All-time playtime Gist ID
	gistIDRecent := os.Getenv("GIST_ID_RECENT") // Recent playtime Gist ID

	multiLined := os.Getenv("MULTILINE") == "YES"

	updateOption := os.Getenv("GIST") 			// options for update: GIST, MARKDOWN, GIST_AND_MARKDOWN
	markdownFile := os.Getenv("MARKDOWN_FILE")

	var updateGist, updateMarkdown bool
	if updateOption == "MARKDOWN" {
		updateMarkdown = true
	} else if updateOption == "GIST_AND_MARKDOWN" {
		updateGist = true
		updateMarkdown = true
	} else {
		updateGist = true
	}

	box := steambox.NewBox(steamAPIKey, ghUsername, ghToken)

	ctx := context.Background()

	// Update all-time playtime Gist
	if gistID != "" {
		var lines []string
		var err error
		for retries := 0; retries < 5; retries++ {
			lines, err = box.GetPlayTime(ctx, steamID, multiLined, appIDList...)
			if err == nil {
				break
			}
			if strings.Contains(err.Error(), "429") {
				wait := time.Duration(2<<retries) * time.Second
				fmt.Printf("Received 429. Retrying in %v...\n", wait)
				time.Sleep(wait)
				continue
			}
			panic("GetPlayTime error: " + err.Error())
		}

		if err != nil {
			panic("GetPlayTime failed after retries: " + err.Error())
		}

		if updateGist {
			updateGistContent(ctx, box, gistID, "⭐ My Most Played Steam Games", lines)
		}

		if updateMarkdown && markdownFile != "" {
			updateMarkdownFile(ctx, markdownFile, gistID, "⭐ My Most Played Steam Games", lines)
		}
	}

	// Update recent playtime Gist
	if gistIDRecent != "" {
		var lines []string
		var err error
		for retries := 0; retries < 5; retries++ {
			lines, err = box.GetRecentGames(ctx, steamID, multiLined)
			if err == nil {
				break
			}
			if strings.Contains(err.Error(), "429") {
				wait := time.Duration(2<<retries) * time.Second
				fmt.Printf("Received 429 on recent games. Retrying in %v...\n", wait)
				time.Sleep(wait)
				continue
			}
			panic("GetRecentGames error: " + err.Error())
		}
		if err != nil {
			panic("GetRecentGames failed after retries: " + err.Error())
		}

		if updateGist {
			updateGistContent(ctx, box, gistIDRecent, "🔥 Recently Played Steam Games", lines)
		}

		if updateMarkdown && markdownFile != "" {
			updateMarkdownFile(ctx, markdownFile, gistIDRecent, "🔥 Recently Played Steam Games", lines)
		}
	}
}

// Helper function to update Gist content
func updateGistContent(ctx context.Context, box *steambox.Box, gistID, filename string, lines []string) {
	gist, err := box.GetGist(ctx, gistID)
	if err != nil {
		panic("GetGist error: " + err.Error())
	}

	f := gist.Files[github.GistFilename(filename)]
	f.Content = github.String(strings.Join(lines, "\n"))
	gist.Files[github.GistFilename(filename)] = f

	err = box.UpdateGist(ctx, gistID, gist)
	if err != nil {
		panic("UpdateGist error: " + err.Error())
	}
	fmt.Println("Updated Gist:", filename)
}

// Helper function to update Markdown file
func updateMarkdownFile(ctx context.Context, markdownFile, gistID, title string, lines []string) {
	titleLink := fmt.Sprintf(`#### <a href="https://gist.github.com/%s" target="_blank">%s</a>`, gistID, title)

	content := bytes.NewBuffer(nil)
	content.WriteString(strings.Join(lines, "\n"))

	box := steambox.NewBox(os.Getenv("STEAM_API_KEY"), os.Getenv("GH_USER"), os.Getenv("GH_TOKEN"))
	err := box.UpdateMarkdown(ctx, titleLink, markdownFile, content.Bytes())
	if err != nil {
		fmt.Println("UpdateMarkdown error:", err)
	} else {
		fmt.Println("Updated markdown file:", markdownFile)
	}
}
