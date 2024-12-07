package trie

import (
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
	fileName string
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

// Insert inserts a word to the trie.
func (t *Trie) Insert(fileName string) error {
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
	current.fileName = fileName

	return nil
}

func removeNonAlpha(fileName string) string {
	s := strings.ToLower(fileName)

	s = strings.ReplaceAll(s, ".md", "")
	s = strings.ReplaceAll(s, ".org", "")

	return strings.Map(func(r rune) rune {
        if unicode.IsLetter(r) {
            return r
        }
        return -1  // -1 tells strings.Map to remove this rune
    }, s)
}

// SearchWord will return false if the word is not in the trie
// and true if it is in th trie.
func (t *Trie) SearchWord(word string) bool {
	var (
		strippedWord = removeNonAlpha(word)
		current      = t.RootNode
	)

	for i := 0; i < len(strippedWord); i++ {
		index := strippedWord[i] - 'a'

		// When nil this is the last node and this word is not indexed in the trie
		if current == nil {
			return false
		}

		if current.Children[index] == nil {
			return false
		}

		// Move to the next node
		current = current.Children[index]
	}

	// Only return as found if it's an indexed word
	return current.IsEndOfWord
}
