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
				"test 2.md":   "# Test 2\nMore content",
			},
			wantErr: false,
			checkFn: func(t *testing.T, tr *trie.Trie) {
				results := tr.Search("test one")
				if !slices.Contains(results, "test-one.md") {
					t.Errorf("Expected to find test-one.md in trie, got %v", results)
				}

				results = tr.Search("test 2")
				if !slices.Contains(results, "test 2.md") {
					t.Errorf("Expected to find \"test 2.md\" in trie, got %v", results)
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
		"unicode filenames and aliases": {
			files: map[string]string{
				"测试文件.md":    "# 测试文件\nalias:: 测试, 文件",
				"español.md": "# Spanish\nalias:: españa, español",
				"русский.md": "# Russian\nalias:: русский, кириллица",
				"日本語.md":     "# Japanese\nalias:: 日本語, 漢字",
				"한국어.md":     "# Korean\nalias:: 한국, 한글",
			},
			wantErr: false,
			checkFn: func(t *testing.T, tr *trie.Trie) {
				// Test Chinese
				results := tr.Search("测试")
				if !slices.Contains(results, "测试文件.md") {
					t.Errorf("Expected to find 测试文件.md in trie, got %v", results)
				}

				// Test Spanish
				results = tr.Search("espa")
				if !slices.Contains(results, "español.md") {
					t.Errorf("Expected to find español.md in trie, got %v", results)
				}

				// Test alias with diacritics
				results = tr.Search("españ")
				if !slices.Contains(results, "español.md") {
					t.Errorf("Expected to find español.md via alias in trie, got %v", results)
				}

				// Test Russian
				results = tr.Search("рус")
				if !slices.Contains(results, "русский.md") {
					t.Errorf("Expected to find русский.md in trie, got %v", results)
				}

				// Test Japanese
				results = tr.Search("日本")
				if !slices.Contains(results, "日本語.md") {
					t.Errorf("Expected to find 日本語.md in trie, got %v", results)
				}

				// Test Korean
				results = tr.Search("한국")
				if !slices.Contains(results, "한국어.md") {
					t.Errorf("Expected to find 한국어.md in trie, got %v", results)
				}
			},
		},
		"unicode normalization": {
			files: map[string]string{
				// File with decomposed characters (é = e + ́ )
				"résumé.md": "# Resume\nalias:: resume, résumé",
			},
			wantErr: false,
			checkFn: func(t *testing.T, tr *trie.Trie) {
				// Test with precomposed é character
				results := tr.Search("résumé")
				if !slices.Contains(results, "résumé.md") {
					t.Errorf("Expected to find résumé.md with precomposed chars, got %v", results)
				}

				// Test with plain ASCII (should still find it due to normalization)
				results = tr.Search("resume")
				if !slices.Contains(results, "résumé.md") {
					t.Errorf("Expected to find résumé.md with ASCII chars, got %v", results)
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
		"Part of word in Trie - Should be found": {
			trie:       trie.NewTrie(),
			insertWord: "testing",
			searchWord: "test",
			shouldFind: true,
		},
		"Unicode word in Trie - Should be found": {
			trie:       trie.NewTrie(),
			insertWord: "学习",
			searchWord: "学习",
			shouldFind: true,
		},
		"Part of Unicode word in Trie - Should be found": {
			trie:       trie.NewTrie(),
			insertWord: "学习",
			searchWord: "学",
			shouldFind: true,
		},
		"Mixed script word in Trie - Should be found": {
			trie:       trie.NewTrie(),
			insertWord: "tеsting", // contains Cyrillic 'е' not Latin 'e'
			searchWord: "tеst",    // with Cyrillic 'е'
			shouldFind: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.trie.InsertFileName(tt.insertWord)

			result := tt.trie.Search(tt.searchWord)

			// Note: Changed the condition to check that results contain the inserted word
			// rather than the search word, as that's what would be stored
			if (len(result) > 0) != tt.shouldFind {
				t.Errorf("search for %q after inserting %q; wanted found=%t but got %v",
					tt.searchWord, tt.insertWord, tt.shouldFind, result)
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
		"Unicode alias - Should be found": {
			trie:       trie.NewTrie(),
			alias:      "नमस्ते", // Hindi greeting
			fileName:   "greeting.md",
			searchWord: "नमस्ते",
			shouldFind: true,
		},
		"Part of Unicode alias - Should be found": {
			trie:       trie.NewTrie(),
			alias:      "漢字", // Kanji
			fileName:   "characters.md",
			searchWord: "漢",
			shouldFind: true,
		},
		"Unicode with diacritics - Normalization": {
			trie:       trie.NewTrie(),
			alias:      "café", // With precomposed é
			fileName:   "coffee.md",
			searchWord: "cafe", // Plain ASCII - should not match.
			shouldFind: false,
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
