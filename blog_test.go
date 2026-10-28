package main

import (
	"math/rand"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBlogPostsParse(t *testing.T) {
	for _, post := range blogPosts {
		blocks, _ := parseBlogMarkdown(post.Slug, post.Body())
		if len(blocks) < 10 {
			t.Fatalf("%s: only %d blocks, is the markdown embedded?", post.Slug, len(blocks))
		}
		for _, b := range blocks {
			if b.kind == blogCode || b.kind == blogFence {
				continue
			}
			text := string(b.text)
			for _, leftover := range []string{"](", "<br", "**", "[[", "| ---"} {
				if strings.Contains(text, leftover) {
					t.Errorf("%s: unrendered markdown %q in %q", post.Slug, leftover, text)
				}
			}
		}
	}
}

func TestBlogWrapFitsWidth(t *testing.T) {
	for _, post := range blogPosts {
		blocks, _ := parseBlogMarkdown(post.Slug, post.Body())
		for _, width := range []int{10, 37, 83} {
			for _, l := range wrapBlogBlocks(blocks, width) {
				if len(l.text) > width {
					t.Fatalf("%s width %d: line too long %q", post.Slug, width, string(l.text))
				}
				if len(l.attrs) != len(l.text) || len(l.hrefs) != len(l.text) {
					t.Fatalf("%s: attrs/text length mismatch on %q", post.Slug, string(l.text))
				}
			}
		}
	}
}

func blogKey(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "ctrl+d":
		return tea.KeyMsg{Type: tea.KeyCtrlD}
	case "ctrl+u":
		return tea.KeyMsg{Type: tea.KeyCtrlU}
	case "ctrl+f":
		return tea.KeyMsg{Type: tea.KeyCtrlF}
	case "ctrl+b":
		return tea.KeyMsg{Type: tea.KeyCtrlB}
	case "ctrl+e":
		return tea.KeyMsg{Type: tea.KeyCtrlE}
	case "ctrl+y":
		return tea.KeyMsg{Type: tea.KeyCtrlY}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestBatchedKeysMatchSingleKeys(t *testing.T) {
	open := func() tea.Model {
		m := initialModel()
		m.width, m.height = 100, 30
		m.openBlogPost(0)
		return m
	}

	single := open()
	for _, k := range []string{"5", "j", "3", "w", "g", "g", "}"} {
		single, _ = single.Update(blogKey(k))
	}
	batched := open()
	batched, _ = batched.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5j3wgg}")})

	s, b := single.(model).reader, batched.(model).reader
	if s.cy != b.cy || s.cx != b.cx || s.cy == 0 {
		t.Fatalf("batched keys landed at %d:%d, single keys at %d:%d", b.cy, b.cx, s.cy, s.cx)
	}
}

func TestBlogReaderKeysDoNotPanic(t *testing.T) {
	keys := strings.Fields("h j k l w b e W B E 0 ^ $ G g z t f F T ; , { } H M L n N * # / ? : 5 1 x a q enter esc backspace ctrl+d ctrl+u ctrl+f ctrl+b ctrl+e ctrl+y")
	rng := rand.New(rand.NewSource(1))
	for i := range blogPosts {
		for _, size := range [][2]int{{20, 8}, {50, 15}, {120, 40}} {
			m := initialModel()
			m.width, m.height = size[0], size[1]
			m.openBlogPost(i)
			var tm tea.Model = m
			for n := 0; n < 3000; n++ {
				if tm.(model).currentPage != blogPostPage {
					next := tm.(model)
					next.openBlogPost(i)
					tm = next
				}
				if n%500 == 0 {
					tm, _ = tm.Update(tea.WindowSizeMsg{Width: 20 + rng.Intn(120), Height: 8 + rng.Intn(40)})
				}
				tm, _ = tm.Update(blogKey(keys[rng.Intn(len(keys))]))
				_ = tm.View()
			}
		}
	}
}
