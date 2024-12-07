package trie_test

import (
	"testing"

	"github.com/jrswab/lsq/trie"
)

func TestTrieWordSearch(t *testing.T) {
	tests := map[string]struct {
		trie       *trie.Trie
		insertWord string
		searchWord string
		shouldFind bool
	}{
		"Word in Trie - Should be found": {
			trie:       trie.NewTrie(),
			insertWord: "testing",
			searchWord: "testing",
			shouldFind: true,
		},
		"Word Not in Trie - Should not be found": {
			trie:       trie.NewTrie(),
			insertWord: "test",
			searchWord: "testing",
			shouldFind: false,
		},
		"Part of word in Trie - Should not be found": {
			trie:       trie.NewTrie(),
			insertWord: "testing",
			searchWord: "test",
			shouldFind: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := tt.trie.Insert(tt.insertWord)
			if err != nil {
				t.Errorf("Unable to insert %q into trie; err: %s", tt.searchWord, err)
			}

			found := tt.trie.SearchWord(tt.searchWord)
			if found != tt.shouldFind {
				t.Errorf("trie.SearchWord(%q) = %t, want %t", tt.searchWord, found, tt.shouldFind)
			}
		})
	}
}
