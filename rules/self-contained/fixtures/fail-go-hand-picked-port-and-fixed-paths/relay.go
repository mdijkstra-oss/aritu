package scenario

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
)

type Message struct {
	ID   string `json:"id"`
	Body string `json:"body"`
}

type Config struct {
	Addr       string
	SpoolPath  string
	ArchiveDir string
}

type Relay struct {
	config   Config
	listener net.Listener
}

func New(config Config) *Relay {
	return &Relay{config: config}
}

func (r *Relay) Listen() error {
	listener, err := net.Listen("tcp", r.config.Addr)
	if err != nil {
		return err
	}
	r.listener = listener
	go r.drain(listener)
	return nil
}

func (r *Relay) Close() error {
	if r.listener == nil {
		return nil
	}
	return r.listener.Close()
}

func (r *Relay) Spool(message Message) error {
	line, err := json.Marshal(message)
	if err != nil {
		return err
	}

	spool, err := os.OpenFile(r.config.SpoolPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer spool.Close()

	_, err = spool.Write(append(line, '\n'))
	return err
}

func (r *Relay) Archive() error {
	if err := os.MkdirAll(r.config.ArchiveDir, 0o755); err != nil {
		return err
	}
	return os.Rename(r.config.SpoolPath, filepath.Join(r.config.ArchiveDir, archiveFileName))
}

func (r *Relay) drain(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		conn.Close()
	}
}

const archiveFileName = "messages.jsonl"
