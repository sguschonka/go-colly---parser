
# Wikipedia Links Scraper

Этот проект написан на Go и предназначен для парсинга ссылок с указанных страниц Википедии. Результаты сохраняются в Excel-файл `scraped_links.xlsx`, а логирование ведется в терминал и файл `parser.log`.

## 🔧 Как работает

- Использует библиотеку [Colly](https://github.com/gocolly/colly) для парсинга страниц.
- Получает заголовок каждой страницы.
- Собирает все ссылки с каждой страницы.
- Формирует Excel-таблицу с результатами.

## 🚀 Как запустить

1. Установите зависимости:
   ```bash
   go mod init scraper
   go get github.com/gocolly/colly/v2
   go get github.com/xuri/excelize/v2
   ```

2. Скомпилируйте и запустите проект:
   ```bash
   go run main.go
   ```

3. Результаты будут сохранены в файл `scraped_links.xlsx`.

## 📦 Выходные файлы

- `parser.log` — логи работы парсера.
- `scraped_links.xlsx` — Excel-файл с тремя колонками:
  - Page URL
  - Page Title
  - Link URL

## ⚙️ Настройка

- Список страниц для парсинга задается в слайсе `urls` в файле `main.go`.
- Параметры Colly (количество параллельных запросов, задержка) настраиваются через `c.Limit()`.


---

* Время выполнения скрипта
![image](https://github.com/user-attachments/assets/b34f38a8-d11a-45c6-8c28-6fff7dce12ad)

---

## Подробное описание кода

### 1️⃣ Импорт библиотек

```go
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
```

* **fmt** — для форматированного вывода.
* **io, os** — для работы с файлами (логами).
* **log** — для логирования.
* **sync** — для синхронизации потоков.
* **time** — для задания задержки между запросами.
* **colly** — основной инструмент парсинга сайтов.
* **excelize** — для создания Excel-файла с результатами.

---

### 2️⃣ Тип данных для хранения ссылок

```go
type Link struct {
	PageURL   string
	PageTitle string
	LinkURL   string
}
```

* Структура `Link` описывает каждую найденную ссылку: с URL страницы, заголовком и самой ссылкой.

---

### 3️⃣ Настройка логирования

```go
file, err := os.OpenFile("parser.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
...
mw := io.MultiWriter(os.Stdout, file)
logger := log.New(mw, "scraper: ", log.LstdFlags)
```

* Логи пишутся одновременно в файл `parser.log` и выводятся в терминал через `MultiWriter`.
* Если лог-файл не удалось открыть — программа аварийно завершится.

---

### 4️⃣ Задаем список URL для парсинга

```go
urls := []string{
	"https://en.wikipedia.org/wiki/Dota_2",
	...
}
```

* Здесь перечислены страницы Википедии, которые нужно обработать.

---

### 5️⃣ Инициализация Colly

```go
c := colly.NewCollector(
	colly.Async(true),
	colly.UserAgent("..."),
)
```

* Включена асинхронность, чтобы загружать несколько страниц параллельно.
* p.s. асинхронность в colly создается при помощи горутин, все что нужно - довериться ей
* Указан User-Agent для имитации реального браузера.

---

### 6️⃣ Ограничения Colly

```go
c.Limit(&colly.LimitRule{
	DomainGlob:  "*",
	Parallelism: 3,
	Delay:       10 * time.Millisecond,
})
```

* Ограничение на 3 параллельных запроса с минимальной задержкой 10 мс между ними.

---

### 7️⃣ Подготовка общих переменных

```go
var allLinks []Link
var titles = make(map[string]string)
var mu sync.Mutex
```

* `allLinks` — слайс для всех найденных ссылок.
* `titles` — карта для хранения заголовков страниц (ключ — URL страницы).
* `mu` — мьютекс для синхронизации доступа к общим данным при многопоточности.

---

### 8️⃣ Основные обработчики Colly

#### Логирование каждого запроса:

```go
c.OnRequest(func(r *colly.Request) {
	logger.Println("Посещаем:", r.URL)
})
```

#### Сбор заголовков страниц:

```go
c.OnHTML("h1#firstHeading", func(e *colly.HTMLElement) {
	mu.Lock()
	titles[e.Request.URL.String()] = e.ChildText("i")
	mu.Unlock()
	logger.Println("Заголовок:", e.ChildText("i"))
})
```

* Заголовок страницы ищется по `h1#firstHeading i` и сохраняется в карту.

#### Сбор всех ссылок на странице:

```go
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
```

* Извлекаются ссылки внутри блока с классом `mw-body-content`.
* Преобразуются в абсолютные URL и добавляются в слайс `allLinks`.

#### Обработка ошибок:

```go
c.OnError(func(r *colly.Response, err error) {
	fmt.Println("Ошибка:", err)
})
```

---

### 9️⃣ Старт парсинга и ожидание завершения

```go
for _, url := range urls {
	err := c.Visit(url)
	if err != nil {
		logger.Printf("Не удалось посетить %s: %v\n", url, err)
	}
}
c.Wait()
```

* Для каждой страницы из `urls` вызывается `Visit`.
* `c.Wait()` дожидается завершения всех асинхронных задач.

---

### 🔟 Постобработка ссылок

```go
for i, link := range allLinks {
	mu.Lock()
	if title, exists := titles[link.PageURL]; exists {
		allLinks[i].PageTitle = title
	} else {
		allLinks[i].PageTitle = "Неизвестный заголовок"
	}
	mu.Unlock()
}
```

* После парсинга каждому элементу `allLinks` присваивается актуальный заголовок страницы или "Неизвестный заголовок", если его не нашли.

---

### 🔥 Вывод количества ссылок

```go
logger.Printf("Количество ссылок: %d", len(allLinks))
```

---

### 📊 Создание Excel-файла

```go
f := excelize.NewFile()
defer f.Close()
sheet := "Результаты"
f.SetSheetName("Sheet1", sheet)
```

* Создается новый файл Excel и переименовывается лист.

---

### 📝 Заполнение Excel-таблицы

1. Заголовки столбцов:

```go
f.SetCellValue(sheet, "A1", "Page URL")
f.SetCellValue(sheet, "B1", "Page Title")
f.SetCellValue(sheet, "C1", "Link URL")
```

2. Данные:

```go
for i, link := range allLinks {
	f.SetCellValue(sheet, fmt.Sprintf("A%d", i+2), link.PageURL)
	f.SetCellValue(sheet, fmt.Sprintf("B%d", i+2), link.PageTitle)
	f.SetCellValue(sheet, fmt.Sprintf("C%d", i+2), link.LinkURL)
}
```

---

### 💾 Сохранение Excel-файла

```go
if err := f.SaveAs("scraped_links.xlsx"); err != nil {
	logger.Fatalf("Ошибка при сохранении Excel-файла: %v", err)
}
logger.Println("Файл успешно сохранен")
```

---
