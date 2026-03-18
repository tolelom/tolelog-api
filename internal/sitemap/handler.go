package sitemap

import (
	"encoding/xml"
	"fmt"
	"time"
	"tolelom_api/internal/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const siteURL = "https://tolelom.xyz"

// XML Sitemap structs (sitemaps.org protocol)

type urlSet struct {
	XMLName xml.Name  `xml:"urlset"`
	XMLNS   string    `xml:"xmlns,attr"`
	URLs    []siteURL_ `xml:"url"`
}

type siteURL_ struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// sitemapEntry holds minimal data for sitemap generation.
type sitemapEntry struct {
	ID        uint
	UpdatedAt time.Time
}

// Sitemap godoc
// @Summary      XML 사이트맵
// @Description  공개 글과 시리즈의 URL을 XML 사이트맵 형식으로 반환합니다
// @Tags         SEO
// @Produce      xml
// @Success      200  {string}  string  "XML Sitemap"
// @Router       /sitemap.xml [get]
func (h *Handler) Sitemap(c *fiber.Ctx) error {
	urls := []siteURL_{
		{Loc: siteURL, LastMod: time.Now().Format(time.DateOnly)},
	}

	// Public posts
	var posts []sitemapEntry
	if err := h.db.Model(&model.Post{}).
		Select("id, updated_at").
		Where("is_public = ? AND deleted_at IS NULL", true).
		Order("updated_at DESC").
		Find(&posts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("사이트맵 생성에 실패했습니다")
	}

	for _, p := range posts {
		urls = append(urls, siteURL_{
			Loc:     fmt.Sprintf("%s/post/%d", siteURL, p.ID),
			LastMod: p.UpdatedAt.Format(time.DateOnly),
		})
	}

	// Series
	var seriesList []sitemapEntry
	if err := h.db.Model(&model.Series{}).
		Select("id, updated_at").
		Where("deleted_at IS NULL").
		Order("updated_at DESC").
		Find(&seriesList).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("사이트맵 생성에 실패했습니다")
	}

	for _, s := range seriesList {
		urls = append(urls, siteURL_{
			Loc:     fmt.Sprintf("%s/series/%d", siteURL, s.ID),
			LastMod: s.UpdatedAt.Format(time.DateOnly),
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
