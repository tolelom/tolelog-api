package feed

import (
	"encoding/xml"
	"time"
	"tolelom_api/internal/post"
	"tolelom_api/internal/utils"

	"github.com/gofiber/fiber/v2"
)

const (
	rssItemLimit  = 20
	siteURL       = "https://tolelom.xyz"
	blogTitle     = "Tolelog"
	blogDesc      = "Tolelog 블로그"
	excerptLength = 200
)

// RSS 2.0 structs

type rssRoot struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Items         []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Author      string `xml:"author,omitempty"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

type Handler struct {
	postService post.Service
}

func NewHandler(postService post.Service) *Handler {
	return &Handler{postService: postService}
}

func excerpt(content string, maxLen int) string {
	plain := utils.StripMarkdown(content)
	if len([]rune(plain)) <= maxLen {
		return plain
	}
	return string([]rune(plain)[:maxLen]) + "..."
}

// Feed godoc
// @Summary      RSS 피드
// @Description  공개 글의 RSS 2.0 피드를 반환합니다 (최근 20개)
// @Tags         Feed
// @Produce      xml
// @Success      200  {string}  string  "RSS 2.0 XML"
// @Router       /feed [get]
func (h *Handler) Feed(c *fiber.Ctx) error {
	posts, _, err := h.postService.GetPublicPosts(1, rssItemLimit, "")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("피드 생성에 실패했습니다")
	}

	items := make([]rssItem, 0, len(posts))
	for _, p := range posts {
		author := ""
		if p.User.ID != 0 {
			author = p.User.Username
		}
		items = append(items, rssItem{
			Title:       p.Title,
			Link:        siteURL + "/post/" + uintToStr(p.ID),
			Description: excerpt(p.Content, excerptLength),
			Author:      author,
			PubDate:     p.CreatedAt.Format(time.RFC1123Z),
			GUID:        siteURL + "/post/" + uintToStr(p.ID),
		})
	}

	lastBuild := time.Now().Format(time.RFC1123Z)
	if len(posts) > 0 {
		lastBuild = posts[0].CreatedAt.Format(time.RFC1123Z)
	}

	rss := rssRoot{
		Version: "2.0",
		Channel: rssChannel{
			Title:         blogTitle,
			Link:          siteURL,
			Description:   blogDesc,
			Language:      "ko",
			LastBuildDate: lastBuild,
			Items:         items,
		},
	}

	output, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("피드 생성에 실패했습니다")
	}

	c.Set("Content-Type", "application/rss+xml; charset=utf-8")
	return c.Send(append([]byte(xml.Header), output...))
}

func uintToStr(n uint) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
