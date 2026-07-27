package scenario

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

type Entry struct {
	ID   string `json:"id"`
	Body string `json:"body"`
}

type Journal struct {
	path string
}

func Open(dir string) (*Journal, error) {
	path := filepath.Join(dir, logFileName)
	log, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	if err := log.Close(); err != nil {
		return nil, err
	}
	return &Journal{path: path}, nil
}

func (j *Journal) Append(entry Entry) error {
	log, err := os.OpenFile(j.path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer log.Close()

	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = log.Write(append(line, '\n'))
	return err
}

func (j *Journal) Entries() ([]Entry, error) {
	log, err := os.Open(j.path)
	if err != nil {
		return nil, err
	}
	defer log.Close()

	entries := []Entry{}
	lines := bufio.NewScanner(log)
	for lines.Scan() {
		var entry Entry
		if err := json.Unmarshal(lines.Bytes(), &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, lines.Err()
}

const logFileName = "entries.log"
