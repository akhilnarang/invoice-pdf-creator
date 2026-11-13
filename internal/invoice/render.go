package invoice

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RenderHTMLFile renders the invoice to an HTML file at outPath.
func (inv Invoice) RenderHTMLFile(outPath string) (err error) {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
	}()

	w := bufio.NewWriterSize(f, len(templateHTML))
	if err := inv.renderHTML(w); err != nil {
		return err
	}
	return w.Flush()
}

// safeFileName keeps only characters safe for a filename, so an invoice number
// can be used verbatim as the temp HTML name (which becomes the tab title).
func safeFileName(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_' || r == '.':
			return r
		default:
			return '_'
		}
	}, s)
}

// fileURL turns an absolute path into an escaped file:// URL, so characters
// like '#' or '?' in the temp dir stay part of the path.
func fileURL(abs string) string {
	p := filepath.ToSlash(abs)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p // Windows: C:/x -> /C:/x
	}
	return (&url.URL{Scheme: "file", Path: p}).String()
}

// RenderPDF renders the invoice to a PDF at outPath using headless Chrome.
// chromeBin is the Chrome/Chromium executable (name on PATH or absolute path).
func (inv Invoice) RenderPDF(outPath, chromeBin string) error {
	// Use a unique temp dir but a clean, invoice-named file inside it, so the
	// browser tab / PDF title reads e.g. "INV-0002-03.html" rather than a
	// random temp name.
	dir, err := os.MkdirTemp("", "invoice-render-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	name := safeFileName(inv.Number)
	if name == "" {
		name = "invoice"
	}
	htmlPath := filepath.Join(dir, name+".html")
	if err := inv.RenderHTMLFile(htmlPath); err != nil {
		return err
	}

	abs, err := filepath.Abs(htmlPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	cmd := exec.Command(chromeBin,
		"--headless", "--disable-gpu",
		"--no-pdf-header-footer",
		"--print-to-pdf="+outPath,
		fileURL(abs),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("chrome (%s) did not make the PDF file: %v\n%s", chromeBin, err, out)
	}
	return nil
}
