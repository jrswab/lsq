package trie

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// Node represents a single character in the trie.
// Char is the string repersentation of the character
// Each node has 26 childe nodes which represent each
// letter of the alphabet.
type Node struct {
	Char        string
	Children    [26]*Node
	IsEndOfWord bool
	IsAlias     bool
	FileName    string
}

// NewNode is used to initialize a new node with it's 26 children
// and each child should first be initialized to nil
func NewNode(char string) *Node {
	node := &Node{Char: char}

	for i := 0; i < 26; i++ {
		node.Children[i] = nil
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
	var tree = NewTrie()

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
		current      = t.RootNode
		strippedWord = removeNonAlpha(fileName)
	)

	for i := 0; i < len(strippedWord); i++ {
		// a is decimal number 97
		// b is decimal number 98
		// and so on....
		// 98-97=1 so the index of b is 1
		index := strippedWord[i] - 'a'

		// Check if the current node has a child node created for this letter (ascii digit)
		// if not create the node:
		if current.Children[index] == nil {
			current.Children[index] = NewNode(string(strippedWord[i]))
		}

		current = current.Children[index]
	}

	// Mark this as end of the word to help avoid false positives.
	current.IsEndOfWord = true
	current.FileName = fileName

	return
}

func (t *Trie) InsertAlias(alias, fileName string) {
	var (
		current      = t.RootNode
		strippedWord = removeNonAlpha(alias)
	)

	for i := 0; i < len(strippedWord); i++ {
		// a is decimal number 97
		// b is decimal number 98
		// and so on....
		// 98-97=1 so the index of b is 1
		index := strippedWord[i] - 'a'

		// Check if the current node has a child node created for this letter (ascii digit)
		// if not create the node:
		if current.Children[index] == nil {
			current.Children[index] = NewNode(string(strippedWord[i]))
		}

		current = current.Children[index]
	}

	// Mark this as end of the word to help avoid false positives.
	current.IsEndOfWord = true
	current.FileName = fileName
	current.IsAlias = true

	return
}

func removeNonAlpha(fileName string) string {
	s := strings.ToLower(fileName)

	s = strings.ReplaceAll(s, ".md", "")
	s = strings.ReplaceAll(s, ".org", "")

	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) {
			return r
		}
		return -1 // -1 tells strings.Map to remove this rune
	}, s)
}

func (t *Trie) Search(prefix string) []string {
	results := make(map[string]struct{}) // Use map for deduplication
	current := t.RootNode

	strippedPrefix := removeNonAlpha(prefix)

	// Navigate to prefix node
	for i := 0; i < len(strippedPrefix); i++ {
		index := strippedPrefix[i] - 'a'
		if current == nil || current.Children[index] == nil {
			return nil
		}
		current = current.Children[index]
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
		if child != nil {
			collectFilesUnique(child, results)
		}
	}
}
