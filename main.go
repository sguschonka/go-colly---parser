package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/xuri/excelize/v2"
)

type Link struct {
	PageURL   string
	PageTitle string
	LinkURL   string
}

func main() {
	//открытие файла для логов
	file, err := os.OpenFile("parser.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Не удалось открыть файл логов: %v", err)
	}
	//настройка multiwriter для одновременной записи и в терминал(logger) и в логи(файл)
	mw := io.MultiWriter(os.Stdout, file)
	logger := log.New(mw, "scraper: ", log.LstdFlags)

	urls := []string{
		"https://en.wikipedia.org/wiki/Dota_2",
		"https://en.wikipedia.org/wiki/Counter-Strike:_Global_Offensive",
		"https://en.wikipedia.org/wiki/Counter-Strike_2",
		"https://en.wikipedia.org/wiki/Counter-Strike_(video_game)",
	}

	c := colly.NewCollector(
		colly.Async(true),
		colly.UserAgent("Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Mobile Safari/537.36"),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 3,
		Delay:       10 * time.Millisecond,
	})

	var allLinks []Link
	var titles = make(map[string]string)
	var mu sync.Mutex

	c.OnRequest(func(r *colly.Request) {
		logger.Println("Посещаем:", r.URL)
	})

	c.OnHTML("h1#firstHeading", func(e *colly.HTMLElement) {
		mu.Lock()
		titles[e.Request.URL.String()] = e.ChildText("i")
		mu.Unlock()
		logger.Println("Заголовок:", e.ChildText("i"))
	})

	c.OnHTML("div.mw-body-content a", func(h *colly.HTMLElement) {
		link := h.Attr("href")
		absoluteLink := h.Request.AbsoluteURL(link)
		mu.Lock()
		if absoluteLink != "" {
			allLinks = append(allLinks, Link{
				PageURL:   h.Request.URL.String(),
				LinkURL:   absoluteLink,
				PageTitle: titles[h.Request.URL.String()],
			})
		}
		mu.Unlock()
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Ошибка:", err)
	})
	for _, url := range urls {
		err := c.Visit(url)
		if err != nil {
			logger.Printf("Не удалось посетить %s: %v\n", url, err)
		}
	}
	c.Wait()
	for i, link := range allLinks {
		mu.Lock()
		if title, exists := titles[link.PageURL]; exists {
			allLinks[i].PageTitle = title
		} else {
			allLinks[i].PageTitle = "Неизвестный заголовок"
		}
		mu.Unlock()
	}
	logger.Printf("Количество ссылок: %d", len(allLinks))

	//создаем excel-файл
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Результаты"
	f.SetSheetName("Sheet1", sheet)

	//устанавливаем заголовки таблицы
	f.SetCellValue(sheet, "A1", "Page URL")
	f.SetCellValue(sheet, "B1", "Page Title")
	f.SetCellValue(sheet, "C1", "Link URL")

	//заполняем таблицы данными
	for i, link := range allLinks {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", i+2), link.PageURL)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", i+2), link.PageTitle)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", i+2), link.LinkURL)
	}

	//сохраняем экселевский файл
	if err := f.SaveAs("scraped_links.xlsx"); err != nil {
		logger.Fatalf("Ошибка при сохранении Excel-файла: %v", err)
	}
	logger.Println("Файл успешно сохранен")
}
