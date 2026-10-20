package api

import (
	"errors"
	"fmt"
)

func isEmpty(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return typed == ""
	case map[string]any:
		return len(typed) == 0
	case []any:
		return len(typed) == 0
	default:
		return false
	}
}

func nonEmptyString(value any) bool {
	text, ok := value.(string)
	return ok && text != ""
}

// validateRecord rejects obviously invalid payloads before calling EdgeX.
func validateRecord(kind string, record map[string]any, creating bool) error {
	if !nonEmptyString(record["name"]) {
		return errors.New("name 不能为空")
	}
	if state, ok := record["adminState"]; ok {
		if state != "LOCKED" && state != "UNLOCKED" {
			return errors.New("adminState 必须为 LOCKED 或 UNLOCKED")
		}
	}
	if state, ok := record["operatingState"]; ok {
		if state != "UP" && state != "DOWN" {
			return errors.New("operatingState 必须为 UP 或 DOWN")
		}
	}
	switch kind {
	case "node":
		if creating {
			for _, field := range []string{"serviceName", "profileName", "protocols"} {
				if isEmpty(record[field]) {
					return fmt.Errorf("%s 不能为空", field)
				}
			}
		}
		if value, ok := record["protocols"]; ok {
			if _, isMap := value.(map[string]any); !isMap {
				return errors.New("protocols 必须为 JSON 对象")
			}
		}
	case "watcher":
		if creating {
			for _, field := range []string{"serviceName", "identifiers", "discoveredNode"} {
				if isEmpty(record[field]) {
					return fmt.Errorf("%s 不能为空", field)
				}
			}
		}
	case "profile":
		for _, field := range []string{"resources", "commands"} {
			if value, ok := record[field]; ok {
				if _, isList := value.([]any); !isList {
					return fmt.Errorf("%s 必须为数组", field)
				}
			}
		}
	}
	return nil
}
