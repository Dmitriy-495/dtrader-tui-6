// Файл settings.go — настройки главного лайаута (пропорции панелей).
//
// Решение из чата: "вынести все пропорции панелей в настройки
// интерфейса (в дальнейшем эту технологию будем совершенствовать)".
// LayoutSettings читается из отдельного файла layout.yaml рядом с
// .env — решение из чата: "отдельный файл layout.yaml — удобно
// редактировать вручную, без пересборки". Перечитывается один раз при
// старте TUI (см. LoadLayoutSettings), без hot-reload — осознанное
// упрощение для первой версии, явно обсуждено в чате ("при
// перезапуске" — не "на лету без перезапуска").
package tui

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// LayoutSettings — пропорции панелей главного лайаута, в процентах
// от соответствующего измерения терминала. Проценты, а не абсолютные
// колонки/строки — тот же лайаут остаётся пропорциональным на разных
// размерах терминала, не только на том, для которого его подбирали.
type LayoutSettings struct {
	// RightbarWidthPercent — доля общей ширины терминала под правую
	// панель (LOGS+POSITIONS). Решение из чата: "увеличь панель логов
	// и позиций по ширине до 40% (на контент остаётся 60%)" — было
	// фиксированные 35 колонок независимо от ширины терминала.
	RightbarWidthPercent int `yaml:"rightbar_width_percent"`

	// PositionsHeightPercent — доля высоты ПРАВОЙ ПАНЕЛИ (не всего
	// терминала), выделяемая под POSITIONS; остаток достаётся LOGS.
	// Решение из чата: "увеличь панель открытых позиций по высоте до
	// 40% (на лог остаётся 60%)".
	PositionsHeightPercent int `yaml:"positions_height_percent"`

	// NewsHeightRows — решение из чата: "новостная лента внизу области
	// контента фиксированной высотой в 10 строк" — это единственная
	// панель, заданная фиксированным числом строк, а не процентом:
	// новостная строка по своей природе не должна "тянуться" вместе с
	// ростом терминала (в отличие от rightbar/positions, где больше
	// места — это больше пользы), нескольким лишним строкам с
	// заголовками новостей высокий терминал ничего не добавит.
	NewsHeightRows int `yaml:"news_height_rows"`
}

// DefaultLayoutSettings — значения по умолчанию, решения из этого
// чата. Именно они использовались при первой реализации main-лайаута,
// и используются, если layout.yaml отсутствует или не задаёт
// какое-то конкретное поле (см. LoadLayoutSettings).
func DefaultLayoutSettings() LayoutSettings {
	return LayoutSettings{
		RightbarWidthPercent:   40,
		PositionsHeightPercent: 40,
		NewsHeightRows:         10,
	}
}

// LoadLayoutSettings читает layout.yaml по указанному пути. Если файл
// не существует — возвращает DefaultLayoutSettings() без ошибки (тот
// же принцип, что config.Load() для .env: отсутствие необязательного
// файла настроек — не повод падать). Если файл существует, но
// повреждён (невалидный YAML или значения вне допустимого диапазона) —
// возвращает ошибку явно: битый файл настроек, который пользователь
// сам же и редактирует руками, должен быть замечен при старте, а не
// молча заменён дефолтами.
func LoadLayoutSettings(path string) (LayoutSettings, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultLayoutSettings(), nil
	}
	if err != nil {
		return LayoutSettings{}, fmt.Errorf("чтение %s: %w", path, err)
	}

	cfg := DefaultLayoutSettings() // старт с дефолтов — частично заполненный YAML не обнуляет остальные поля
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return LayoutSettings{}, fmt.Errorf("разбор %s: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return LayoutSettings{}, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

func (s LayoutSettings) validate() error {
	if s.RightbarWidthPercent <= 0 || s.RightbarWidthPercent >= 100 {
		return fmt.Errorf("rightbar_width_percent должен быть в диапазоне 1-99, получено %d", s.RightbarWidthPercent)
	}
	if s.PositionsHeightPercent <= 0 || s.PositionsHeightPercent >= 100 {
		return fmt.Errorf("positions_height_percent должен быть в диапазоне 1-99, получено %d", s.PositionsHeightPercent)
	}
	if s.NewsHeightRows < 0 {
		return fmt.Errorf("news_height_rows не может быть отрицательным, получено %d", s.NewsHeightRows)
	}
	return nil
}

// rightbarWidth возвращает ширину правой панели в колонках для
// заданной полной ширины терминала — округление вниз (int()), чтобы
// никогда не запросить больше места, чем реально есть.
func (s LayoutSettings) rightbarWidth(totalWidth int) int {
	w := totalWidth * s.RightbarWidthPercent / 100
	if w < 1 {
		w = 1
	}
	return w
}

// positionsHeight возвращает высоту блока POSITIONS в строках для
// заданной полной высоты правой панели (rightbar), округление вниз.
func (s LayoutSettings) positionsHeight(rightbarTotalHeight int) int {
	h := rightbarTotalHeight * s.PositionsHeightPercent / 100
	if h < 1 {
		h = 1
	}
	return h
}
