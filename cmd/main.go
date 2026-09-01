// Точка входа dtrader-tui-6 — главный лайаут (App), а не одиночная
// вкладка символа (см. историю в чате: MVP-шаги 1-2 использовали
// symbolMVP как временный захардкоженный символ; App заменяет это
// полноценным лайаутом с автовкладками по symbols из system-канала).
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dmitriy-495/dtrader-tui-6/internal/config"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/tui"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/ws"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Ошибка загрузки конфига: %v", err)
	}

	// Перенаправляем стандартный log (используется пакетом ws —
	// подключено/ошибки/реконнект) в файл вместо stderr — решение из
	// чата: без этого log.Printf из фоновой горутины client.Run
	// печатался прямо в терминал поверх/до/после bubbletea-рендера
	// (AltScreen переключает буфер терминала, но не перехватывает
	// произвольную запись в stderr — это два независимых механизма).
	// Временное решение до появления отдельного лог-контейнера в
	// главном лайауте (rightbar.go уже показывает логи внутри TUI —
	// см. App.addLog — но сам пакет ws пока логирует через стандартный
	// log, а не передаёт события в App напрямую; статусы подключения/
	// реконнекта App всё же дублирует в свой rightbar через
	// appWsStatusMsg, так что дублирование в файл — подстраховка на
	// случай ошибок, которые ws.Client логирует, но не сообщает через
	// Status).
	//
	// Открываем именно здесь, ПОСЛЕ проверки конфига (её ошибка
	// должна остаться видна в обычном терминале) и ДО go client.Run(ctx).
	logFile, err := os.OpenFile("dtrader-tui.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("❌ Не удалось открыть лог-файл: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// layout.yaml — пропорции панелей главного лайаута, решение из
	// чата: "вынести все пропорции панелей в настройки интерфейса".
	// Отсутствие файла — не ошибка (DefaultLayoutSettings), а вот
	// файл, который есть, но не парсится или содержит значения вне
	// допустимого диапазона, — это log.Fatalf, а не тихий откат на
	// дефолты: пользователь, который сам редактирует layout.yaml
	// руками, должен сразу увидеть, что опечатался, а не гадать,
	// почему пропорции не изменились.
	layoutSettings, err := tui.LoadLayoutSettings("layout.yaml")
	if err != nil {
		log.Fatalf("❌ Ошибка загрузки layout.yaml: %v", err)
	}

	client := ws.New(cfg.WSServerURL, cfg.WSAPIKey)
	go client.Run(ctx)

	model := tui.NewApp(client, layoutSettings)
	// tea.WithAltScreen: рисуем в отдельном полноэкранном буфере
	// терминала (как vim/htop), а не построчно в обычном выводе.
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Пробрасываем отмену ctx (Ctrl+C/SIGTERM) в остановку bubbletea —
	// иначе программа продолжила бы рисовать экран после того, как
	// WS-клиент уже завершился по сигналу.
	go func() {
		<-ctx.Done()
		p.Quit()
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Ошибка TUI: %v\n", err)
		os.Exit(1)
	}
}
