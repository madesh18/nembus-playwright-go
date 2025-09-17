package playwright

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/go-jose/go-jose/v3/json"
)

type transport interface {
	Send(msg map[string]interface{}) error
	Poll() (*message, error)
	Close() error
}

type pipeTransport struct {
	writer    io.WriteCloser
	bufReader *bufio.Reader
	closed    chan struct{}
	onClose   func() error
}

func (t *pipeTransport) Poll() (*message, error) {
	msg := &message{}
	// Only log metadata, not message content to prevent exposure
	log.Println("pipeTransport polling started")
	log.Println("Version identifier:", myName)
	if t.isClosed() {
		return nil, fmt.Errorf("transport closed")
	}

	var length uint32
	err := binary.Read(t.bufReader, binary.LittleEndian, &length)
	if err != nil {
		return nil, fmt.Errorf("could not read protocol padding: %w", err)
	}

	data := make([]byte, length)
	_, err = io.ReadFull(t.bufReader, data)
	if err != nil {
		return nil, fmt.Errorf("could not read protocol data: %w", err)
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		// Clear data immediately on error
		for i := range data {
			data[i] = 0
		}
		data = nil
		return nil, fmt.Errorf("could not decode json: %w", err)
	}

	// Clear the raw data immediately after successful unmarshaling
	for i := range data {
		data[i] = 0
	}
	data = nil

	if os.Getenv("DEBUGP") != "" {
		// Only show message metadata in debug, not content
		fmt.Fprintf(os.Stdout, "\x1b[33mRECV>\x1b[0m Message ID: %d, Method: %s, GUID: %s\n",
			msg.ID, msg.Method, msg.GUID)
	}
	// Only log metadata, not message content
	log.Printf("Processed message - ID: %d, Method: %s by version: %s", msg.ID, msg.Method, myName)
	return msg, nil
}

type message struct {
	ID     int                    `json:"id"`
	GUID   string                 `json:"guid"`
	Method string                 `json:"method,omitempty"`
	Params map[string]interface{} `json:"params,omitempty"`
	Result map[string]interface{} `json:"result,omitempty"`
	Error  *struct {
		Error Error `json:"error"`
	} `json:"error,omitempty"`
}

func (t *pipeTransport) Send(msg map[string]interface{}) error {
	// Only log metadata, not message content to prevent exposure
	msgID, _ := msg["id"]
	msgMethod, _ := msg["method"]
	log.Printf("Sending message - ID: %v, Method: %v", msgID, msgMethod)
	if t.isClosed() {
		return fmt.Errorf("transport closed")
	}
	msgBytes, err := json.Marshal(msg)

	// Only log size, not content to prevent exposure
	log.Printf("Message marshaled - size: %d bytes", len(msgBytes))
	if err != nil {
		return fmt.Errorf("pipeTransport: could not marshal json: %w", err)
	}
	if os.Getenv("DEBUGP") != "" {
		// Only show message metadata in debug, not content
		fmt.Fprintf(os.Stdout, "\x1b[32mSEND>\x1b[0m Message ID: %v, Method: %v, Size: %d bytes\n",
			msgID, msgMethod, len(msgBytes))
	}

	lengthPadding := make([]byte, 4)
	binary.LittleEndian.PutUint32(lengthPadding, uint32(len(msgBytes)))
	if _, err = t.writer.Write(append(lengthPadding, msgBytes...)); err != nil {
		// Clear data on error
		for i := range msgBytes {
			msgBytes[i] = 0
		}
		msgBytes = nil
		for i := range lengthPadding {
			lengthPadding[i] = 0
		}
		lengthPadding = nil
		return err
	}

	// Clear all data immediately after successful write
	for i := range msgBytes {
		msgBytes[i] = 0
	}
	msgBytes = nil
	for i := range lengthPadding {
		lengthPadding[i] = 0
	}
	lengthPadding = nil
	// Note: We cannot clear the input msg map as it belongs to the caller
	return nil
}
func (t *pipeTransport) Close() error {
	select {
	case <-t.closed:
		return nil
	default:
		return t.onClose()
	}
}

func (t *pipeTransport) isClosed() bool {
	select {
	case <-t.closed:
		return true
	default:
		return false
	}
}

func newPipeTransport(driver *PlaywrightDriver, stderr io.Writer) (transport, error) {
	t := &pipeTransport{
		closed: make(chan struct{}, 1),
	}

	cmd := driver.Command("run-driver")
	cmd.Stderr = stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("could not create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("could not create stdout pipe: %w", err)
	}
	t.writer = stdin
	t.bufReader = bufio.NewReader(stdout)

	t.onClose = func() error {
		select {
		case <-t.closed:
		default:
			close(t.closed)
		}
		if err := t.writer.Close(); err != nil {
			return err
		}
		// playwright-cli will exit when its stdin is closed
		if err := cmd.Wait(); err != nil {
			return err
		}
		return nil
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("could not start driver: %w", err)
	}

	return t, nil
}
