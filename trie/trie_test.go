package trie_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jrswab/lsq/trie"
)

func TestInit(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "logseq-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test cases
	testCases := map[string]struct {
		files   map[string]string
		wantErr bool
		checkFn func(*testing.T, *trie.Trie)
	}{
		"basic files without aliases": {
			files: map[string]string{
				"test-one.md": "# Test 1\nSome content",
				"test two.md": "# Test 2\nMore content",
			},
			wantErr: false,
			checkFn: func(t *testing.T, tr *trie.Trie) {
				results := tr.Search("test one")
				if !slices.Contains(results, "test-one.md") {
					t.Errorf("Expected to find test-one.md in trie, got %v", results)
				}

				results = tr.Search("test2")
				if !slices.Contains(results, "test-one.md") {
					t.Errorf("Expected to find \"test two.md\" in trie, got %v", results)
				}
			},
		},
		"files with aliases": {
			files: map[string]string{
				"test1.md": "# Test 1\nalias:: alias1, alias2",
				"test2.md": "# Test 2\nalias:: alias3",
			},
			wantErr: false,
			checkFn: func(t *testing.T, tr *trie.Trie) {
				results := tr.Search("alias1")
				if len(results) == 0 {
					t.Error("Expected to find alias1 in trie")
				}

				results = tr.Search("alias2")
				if len(results) == 0 {
					t.Error("Expected to find alias2 in trie")
				}

				results = tr.Search("alias3")
				if len(results) == 0 {
					t.Error("Expected to find alias3 in trie")
				}
			},
		},
		"invalid directory": {
			files:   nil,
			wantErr: true,
			checkFn: nil,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			testPath := tempDir
			if name != "invalid directory" {
				// Create test files
				for filename, content := range tc.files {
					filePath := filepath.Join(testPath, filename)
					err := os.WriteFile(filePath, []byte(content), 0644)
					if err != nil {
						t.Fatalf("Failed to create test file: %v", err)
					}
				}
			} else {
				testPath = filepath.Join(tempDir, "nonexistent")
			}

			// Run the test
			tr, err := trie.Init(testPath)

			// Check error expectation
			if (err != nil) != tc.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			// Run additional checks if provided
			if tc.checkFn != nil && err == nil {
				tc.checkFn(t, tr)
			}
		})
	}
}

func TestTrieInsertFileName(t *testing.T) {
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
			tt.trie.InsertFileName(tt.insertWord)

			result := tt.trie.Search(tt.searchWord)

			if !(slices.Contains(result, tt.searchWord) == tt.shouldFind) {
				t.Errorf("alias search; wanted %t trie but trie.Search() returned %t", tt.shouldFind, !tt.shouldFind)
			}
		})
	}
}

func TestTrieInsertAlias(t *testing.T) {
	tests := map[string]struct {
		trie       *trie.Trie
		alias      string
		fileName   string
		searchWord string
		shouldFind bool
	}{
		"Word in Trie - Should be found": {
			trie:       trie.NewTrie(),
			alias:      "testing",
			fileName:   "alias-test-file.md",
			searchWord: "testing",
			shouldFind: true,
		},
		"Word Not in Trie - Should not be found": {
			trie:       trie.NewTrie(),
			alias:      "test",
			fileName:   "alias-test-file.md",
			searchWord: "testing",
			shouldFind: false,
		},
		"Part of word in Trie - Should be found": {
			trie:       trie.NewTrie(),
			alias:      "testing",
			fileName:   "alias-test-file.md",
			searchWord: "test",
			shouldFind: true,
		},
		"wise -> wisdom": {
			trie:       trie.NewTrie(),
			alias:      "wise",
			fileName:   "wisdom.md",
			searchWord: "wise",
			shouldFind: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.trie.InsertAlias(tt.alias, tt.fileName)

			result := tt.trie.Search(tt.searchWord)

			if !(slices.Contains(result, tt.fileName) == tt.shouldFind) {
				t.Errorf("alias search; wanted %t but trie.Search() returned %t", tt.shouldFind, !tt.shouldFind)
			}
		})
	}
}
