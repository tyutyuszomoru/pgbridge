package config

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"pgbridge/internal/logger"
)

type ParseError struct {
	Line    int
	Content string
	Err     error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %v (content: %s)", e.Line, e.Err, e.Content)
}

func LoadConfig(filepath string, log *logger.Logger) (*Config, error) {
	if log != nil {
		log.LogSystemf(logger.LevelInfo, "config", "Loading configuration from: %s", filepath)
	}

	file, err := os.Open(filepath)
	if err != nil {
		if log != nil {
			log.LogConfigError(fmt.Errorf("failed to open config file: %w", err))
		}
		return nil, fmt.Errorf("error on loading config: %w", err)
	}

	defer file.Close()

	config := &Config{
		Databases: []DatabaseConfig{},
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Regex to parse module list: [module1, module2, ...]
	moduleListRegex := regexp.MustCompile(`\[(.*?)\]`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines and comments
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse the line
		dbConfig, err := parseLine(line, moduleListRegex)
		if err != nil {
			parseErr := &ParseError{
				Line:    lineNum,
				Content: line,
				Err:     err,
			}
			if log != nil {
				log.LogConfigError(parseErr)
			}
			return nil, parseErr
		}

		config.Databases = append(config.Databases, *dbConfig)
	}

	if err := scanner.Err(); err != nil {
		scanErr := fmt.Errorf("error reading config file: %w", err)
		if log != nil {
			log.LogConfigError(scanErr)
		}
		return nil, scanErr
	}

	// Validate the entire configuration
	if err := config.Validate(); err != nil {
		validationErr := fmt.Errorf("configuration validation failed: %w", err)
		if log != nil {
			log.LogConfigError(validationErr)
		}
		return nil, validationErr
	}

	// Count total modules
	totalModules := 0
	for _, db := range config.Databases {
		totalModules += len(db.ActiveModules)
	}

	if log != nil {
		log.LogConfigLoaded(len(config.Databases), totalModules)
	}

	return config, nil
}

// parseLine parses a single configuration line
// Format: database_name, connection_string, [module1, module2, ...]
func parseLine(line string, moduleListRegex *regexp.Regexp) (*DatabaseConfig, error) {
	// Find the module list first
	matches := moduleListRegex.FindStringSubmatch(line)
	if len(matches) != 2 {
		return nil, fmt.Errorf("invalid format: module list not found or malformed (expected [module1, module2, ...])")
	}

	moduleListStr := matches[1]

	// Find the position of the opening bracket to split before it
	bracketPos := strings.Index(line, "[")
	if bracketPos == -1 {
		return nil, fmt.Errorf("invalid format: bracket not found")
	}

	// Get the part before the module list and trim any trailing whitespace/commas
	lineBeforeModules := strings.TrimSpace(line[:bracketPos])
	lineBeforeModules = strings.TrimRight(lineBeforeModules, ",")
	lineBeforeModules = strings.TrimSpace(lineBeforeModules)

	// Split by comma to get name and connection string
	// We need to find the first comma for name, and everything else is connection string
	firstCommaPos := strings.Index(lineBeforeModules, ",")
	if firstCommaPos == -1 {
		return nil, fmt.Errorf("invalid format: expected 'name, connection_string, [modules]'")
	}

	name := strings.TrimSpace(lineBeforeModules[:firstCommaPos])
	connStr := strings.TrimSpace(lineBeforeModules[firstCommaPos+1:])

	// Parse modules
	modules := parseModules(moduleListStr)

	if name == "" {
		return nil, fmt.Errorf("database name cannot be empty")
	}

	if connStr == "" {
		return nil, fmt.Errorf("connection string cannot be empty")
	}

	if len(modules) == 0 {
		return nil, fmt.Errorf("no modules specified")
	}

	return &DatabaseConfig{
		Name:             name,
		ConnectionString: connStr,
		ActiveModules:    modules,
	}, nil
}

// parseModules parses the module list string
func parseModules(moduleStr string) []string {
	modules := []string{}

	// Split by comma
	parts := strings.Split(moduleStr, ",")
	for _, part := range parts {
		module := strings.TrimSpace(part)
		if module != "" {
			modules = append(modules, module)
		}
	}

	return modules
}
