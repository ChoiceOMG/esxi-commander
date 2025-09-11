package interactive

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConfirmAction prompts the user to confirm a potentially destructive action
func ConfirmAction(message string) bool {
	return ConfirmActionWithDefault(message, false)
}

// ConfirmActionWithDefault prompts the user to confirm an action with a default response
func ConfirmActionWithDefault(message string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)
	
	var prompt string
	if defaultYes {
		prompt = fmt.Sprintf("%s [Y/n]: ", message)
	} else {
		prompt = fmt.Sprintf("%s [y/N]: ", message)
	}
	
	for {
		fmt.Print(prompt)
		response, err := reader.ReadString('\n')
		if err != nil {
			return false
		}
		
		response = strings.TrimSpace(strings.ToLower(response))
		
		if response == "" {
			return defaultYes
		}
		
		if response == "y" || response == "yes" {
			return true
		}
		
		if response == "n" || response == "no" {
			return false
		}
		
		fmt.Println("Please respond with 'y' or 'n' (or press Enter for default)")
	}
}

// ConfirmDeletion specifically prompts for deletion confirmation
func ConfirmDeletion(resourceType, resourceName string) bool {
	message := fmt.Sprintf("Are you sure you want to delete %s '%s'? This action cannot be undone.", resourceType, resourceName)
	return ConfirmAction(message)
}

// ConfirmBatchOperation prompts for confirmation of batch operations
func ConfirmBatchOperation(operation string, count int, resourceType string) bool {
	message := fmt.Sprintf("This will %s %d %s(s). Continue?", operation, count, resourceType)
	return ConfirmAction(message)
}