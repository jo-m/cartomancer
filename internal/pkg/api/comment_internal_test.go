package api

import (
	"testing"

	"github.com/franiglesias/golden"
)

func TestRenderCommentBody_BasicFormatting(t *testing.T) {
	golden.Verify(t, renderCommentBody("**bold** and *italic*"), golden.Extension(".html"))
}

func TestRenderCommentBody_Lists(t *testing.T) {
	golden.Verify(t, renderCommentBody("- one\n- two\n- three"), golden.Extension(".html"))
}

func TestRenderCommentBody_Strikethrough(t *testing.T) {
	golden.Verify(t, renderCommentBody("~~deleted~~"), golden.Extension(".html"))
}

func TestRenderCommentBody_StripsHeadings(t *testing.T) {
	golden.Verify(t, renderCommentBody("# heading\nparagraph"), golden.Extension(".html"))
}

func TestRenderCommentBody_StripsCodeBlocks(t *testing.T) {
	golden.Verify(t, renderCommentBody("```\ncode\n```"), golden.Extension(".html"))
}

func TestRenderCommentBody_StripsBlockquotes(t *testing.T) {
	golden.Verify(t, renderCommentBody("> quote"), golden.Extension(".html"))
}

func TestRenderCommentBody_LinksRewritten(t *testing.T) {
	golden.Verify(t, renderCommentBody("[click here](https://example.com)"), golden.Extension(".html"))
}

func TestRenderCommentBody_LinkWithSpecialChars(t *testing.T) {
	golden.Verify(t, renderCommentBody("[test](https://example.com/path?a=1&b=2)"), golden.Extension(".html"))
}

func TestRenderCommentBody_MultipleLinks(t *testing.T) {
	golden.Verify(t, renderCommentBody("[a](https://a.com) and [b](https://b.com)"), golden.Extension(".html"))
}

func TestRenderCommentBody_Mixed(t *testing.T) {
	md := `**bold** and *italic* with a [link](https://example.com)

- item one
- item two
`
	golden.Verify(t, renderCommentBody(md), golden.Extension(".html"))
}

func TestRewriteLinks_NoLinks(t *testing.T) {
	golden.Verify(t, rewriteLinks("<p>no links here</p>"), golden.Extension(".html"))
}

func TestRewriteLinks_MultipleLinks(t *testing.T) {
	input := `<p><a href="https://a.com">a</a> and <a href="https://b.com">b</a></p>`
	golden.Verify(t, rewriteLinks(input), golden.Extension(".html"))
}
