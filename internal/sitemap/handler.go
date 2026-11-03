package sitemap

import (
	"encoding/xml"
	"fmt"
	"time"

	"tolelom_api/internal/permalink"

	"github.com/gofiber/fiber/v2"
)

type urlSet struct {
	XMLName xml.Name   `xml:"urlset"`
	XMLNS   string     `xml:"xmlns,attr"`
	URLs    []urlEntry `xml:"url"`
}

type urlEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type Entry struct {
	ID        uint
	Title     string
	UpdatedAt time.Time
}

type Repository interface {
	GetPublicPostEntries() ([]Entry, error)
	GetSeriesEntries() ([]Entry, error)
	GetPublicAuthorEntries() ([]Entry, error)
}

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// @Summary      XML 사이트맵
// @Description  공개 글과 시리즈의 URL을 XML 사이트맵 형식으로 반환합니다
// @Tags         SEO
// @Produce      xml
// @Success      200  {string}  string  "XML Sitemap"
// @Router       /sitemap.xml [get]
func (h *Handler) Sitemap(c *fiber.Ctx) error {
	urls := []urlEntry{
		{Loc: permalink.SiteURL, LastMod: time.Now().Format(time.DateOnly)},
	}

	posts, err := h.repo.GetPublicPostEntries()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("사이트맵 생성에 실패했습니다")
	}
	for _, p := range posts {
		urls = append(urls, urlEntry{
			Loc:     permalink.SiteURL + permalink.PostPath(p.Title, p.ID),
			LastMod: p.UpdatedAt.Format(time.DateOnly),
		})
	}

	seriesList, err := h.repo.GetSeriesEntries()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("사이트맵 생성에 실패했습니다")
	}
	for _, s := range seriesList {
		urls = append(urls, urlEntry{
			Loc:     fmt.Sprintf("%s/series/%d", permalink.SiteURL, s.ID),
			LastMod: s.UpdatedAt.Format(time.DateOnly),
		})
	}

	authors, err := h.repo.GetPublicAuthorEntries()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("사이트맵 생성에 실패했습니다")
	}
	for _, a := range authors {
		urls = append(urls, urlEntry{
			Loc:     fmt.Sprintf("%s/user/%d", permalink.SiteURL, a.ID),
			LastMod: a.UpdatedAt.Format(time.DateOnly),
		})
	}

	set := urlSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	output, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("사이트맵 생성에 실패했습니다")
	}

	c.Set("Content-Type", "application/xml; charset=utf-8")
	return c.Send(append([]byte(xml.Header), output...))
}
