package infrastructure

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

var logger = log.New(os.Stdout, "infrastructure: ", log.LstdFlags)

func fromJson[T any](content string) (T, error) {
    var structType T
	err := json.Unmarshal([]byte(content), &structType)
    if err != nil {
        return structType, fmt.Errorf("fromJson: %w", err)
    }
	return structType, nil
}
