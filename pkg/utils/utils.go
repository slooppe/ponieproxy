package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type SlackRequestBody struct {
}

/*
 * Creates a uniquely named file, based on the host, path and req/res body.
 * The file is named, based on the hashed host, path and request body.
 * This means that any request and it's resposne are named with the same hash.
 * This makes it easy to go through and read them, when opened with "vim *"
 */
func WriteUniqueFile(checksum string, body string, outputDir string, httpDump string, ext string) {
	if outputDir != "./" {
		os.MkdirAll(outputDir, os.ModePerm)
	}

	filePath := fmt.Sprintf("%v/%v.%v", outputDir, checksum, ext)

	if !FileExists(filePath) {
		var constructed string
		if ext == "req" {
			constructed = fmt.Sprintf(`%v %v`, httpDump, body)
		}
		if ext == "res" {
			constructed = fmt.Sprintf(`%v`, httpDump)
		}

		err := AppendToFile(constructed, filePath)
		if err != nil {
			log.Fatalf("error writing to file: %v", err)
		}
	}
}

/*
 * Takes file path and returns lines
 */
func ReadLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

/*
 * Takes data and writes it to a file
 */
func AppendToFile(data string, filePath string) error {
	if filePath != "" {
		f, err := os.OpenFile(filePath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := f.WriteString(data + "\n"); err != nil {
			return err
		}
	}

	return nil
}

/*
 * Check if a file exists
 */
func FileExists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

/*
 * Send a slack notification via webhook
 */
func SendSlackNotification(webhookUrl string, msg string) error {
	slackBody, _ := json.Marshal(struct {
		Text string `json:"text"`
	}{
		Text: msg,
	})

	req, err := http.NewRequest(http.MethodPost, webhookUrl, bytes.NewBuffer(slackBody))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "ok" {
		return errors.New("Non-ok response returned from Slack")
	}
	return nil
}

/*
 * Check if a string contains any word from a list
 */
func ContainsAnyWord(bigString string, words []string) bool {
	for _, w := range words {
		if strings.Contains(bigString, w) {
			fmt.Println(w, "EEEEEEEEEEEEEEEEEEEe")
			return true
		}
	}

	return false
}
