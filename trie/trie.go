package trie

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Node represents a single character in the trie.
// Char is the string representation of the character
// Each child contains a map of a rune to support non-ascii
// characters.
type Node struct {
	Char        string
	Children    map[rune]*Node
	IsEndOfWord bool
	IsAlias     bool
	FileName    string
}

// NewNode is used to initialize a new node with it's 26 children
// and each child should first be initialized to nil
func NewNode(char string) *Node {
	node := &Node{
		Char:     char,
		Children: make(map[rune]*Node),
	}

	return node
}

// Trie is the tree that will hold all of the nodes.
// RootNode is always nil.
type Trie struct {
	RootNode *Node
}

// NewTrie creates a new trie with a root.
// This node is not used to match words so
// it can be anything; ie "\000".
func NewTrie() *Trie {
	return &Trie{RootNode: NewNode("\000")}
}

func Init(path string) (*Trie, error) {
	tree := NewTrie()

	// get list of all files in ~/Logseq/Pages
	fileList, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	// Allocate enough space in case every page has an alias:
	aliases := make(map[string]string, len(fileList))

	// First pass: Process all files and collect aliases
	for i := range fileList {
		if !fileList[i].IsDir() {
			tree.InsertFileName(fileList[i].Name())

			// Get file contents:
			content, err := os.ReadFile(filepath.Join(path, fileList[i].Name()))
			if err != nil {
				return nil, err
			}

			// Split into a slice of lines:
			lines := strings.Split(string(content), "\n")

			// Find the line containing "alias::"
			for _, line := range lines {
				if strings.Contains(line, "alias::") {

					parts := strings.Split(line, "::")
					if len(parts) != 2 {
						continue
					}

					aliasList := strings.Split(parts[1], ",")

					for _, alias := range aliasList {
						trimmedAlias := strings.TrimSpace(alias)

						if trimmedAlias != "" {
							aliases[trimmedAlias] = fileList[i].Name()
						}
					}
				}
			}
		}
	}

	// Second pass: Insert all collected aliases
	for alias, fileName := range aliases {
		tree.InsertAlias(alias, fileName)
	}

	return tree, nil
}

// Insert inserts a word to the trie.
func (t *Trie) InsertFileName(fileName string) {
	var (
		current        = t.RootNode
		normalizedWord = removeNonAlpha(fileName)
	)

	runes := []rune(normalizedWord)

	for _, r := range runes {
		// Check if current node has this rune as a node
		_, exists := current.Children[r]
		if !exists {
			// If not add it
			current.Children[r] = NewNode(string(r))
		}

		current = current.Children[r]
	}

	// Mark this as end of the word to help avoid false positives.
	current.IsEndOfWord = true
	current.FileName = fileName
}

func (t *Trie) InsertAlias(alias, fileName string) {
	current := t.RootNode

	// Use the same normalization function as the other methods
	normalizedAlias := removeNonAlpha(alias)

	// Convert to runes for proper Unicode character handling
	runeAlias := []rune(normalizedAlias)

	for _, r := range runeAlias {
		// Check if the current node has this rune as a child
		_, exists := current.Children[r]
		if !exists {
			// If not, create a new node
			current.Children[r] = NewNode(string(r))
		}

		// Move to the child node
		current = current.Children[r]
	}

	// Mark this as end of the word to help avoid false positives.
	current.IsEndOfWord = true
	current.FileName = fileName
	current.IsAlias = true
}

func removeNonAlpha(fileName string) string {
	// Step 1: Convert to lowercase first
	s := strings.ToLower(fileName)

	// Step 2: Remove file extensions
	s = strings.ReplaceAll(s, ".md", "")
	s = strings.ReplaceAll(s, ".org", "")

	// Step 3: Keep only letter characters
	letters := []rune{}
	for _, r := range s {
		if unicode.IsLetter(r) {
			letters = append(letters, r)
		}
	}
	s = string(letters)

	// NFC = Normalization Form Canonical Composition
	// This ensures that characters like "é" are treated consistently
	// whether they're a single codepoint or "e" + accent mark
	return norm.NFC.String(s)
}

func (t *Trie) Search(prefix string) []string {
	results := make(map[string]struct{}) // Use map for deduplication
	current := t.RootNode

	normalizedPrefix := removeNonAlpha(prefix)

	prefixRunes := []rune(normalizedPrefix)

	for _, r := range prefixRunes {
		if current == nil {
			return nil
		}

		nextNode, exists := current.Children[r]
		if !exists {
			return nil
		}

		current = nextNode
	}

	// Use map for collection
	collectFilesUnique(current, results)

	// Convert map keys to slice
	uniqueResults := make([]string, 0, len(results))
	for fileName := range results {
		uniqueResults = append(uniqueResults, fileName)
	}

	// Sort results for consistent ordering
	sort.Strings(uniqueResults)

	return uniqueResults
}

func collectFilesUnique(node *Node, results map[string]struct{}) {
	if node == nil {
		return
	}

	if node.IsEndOfWord {
		results[node.FileName] = struct{}{}
	}

	for _, child := range node.Children {
		collectFilesUnique(child, results)
	}
}
