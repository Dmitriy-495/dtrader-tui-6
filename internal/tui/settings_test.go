package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLayoutSettings_MissingFileReturnsDefault(t *testing.T) {
	cfg, err := LoadLayoutSettings(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("LoadLayoutSettings() на отсутствующем файле не должен возвращать ошибку: %v", err)
	}
	if cfg != DefaultLayoutSettings() {
		t.Errorf("LoadLayoutSettings() на отсутствующем файле = %+v, ожидался дефолт %+v", cfg, DefaultLayoutSettings())
	}
}

func TestLoadLayoutSettings_ValidFileOverridesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "layout.yaml")
	content := "rightbar_width_percent: 40\npositions_height_percent: 40\nnews_height_rows: 10\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("не удалось создать тестовый файл: %v", err)
	}

	cfg, err := LoadLayoutSettings(path)
	if err != nil {
		t.Fatalf("LoadLayoutSettings() вернул ошибку на валидном файле: %v", err)
	}
	if cfg.RightbarWidthPercent != 40 || cfg.PositionsHeightPercent != 40 || cfg.NewsHeightRows != 10 {
		t.Errorf("LoadLayoutSettings() = %+v, ожидались значения 40/40/10", cfg)
	}
}

func TestLoadLayoutSettings_PartialFileKeepsDefaultsForMissingFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "layout.yaml")
	content := "rightbar_width_percent: 25\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("не удалось создать тестовый файл: %v", err)
	}

	cfg, err := LoadLayoutSettings(path)
	if err != nil {
		t.Fatalf("LoadLayoutSettings() вернул ошибку: %v", err)
	}
	if cfg.RightbarWidthPercent != 25 {
		t.Errorf("RightbarWidthPercent = %d, ожидалось 25", cfg.RightbarWidthPercent)
	}
	def := DefaultLayoutSettings()
	if cfg.PositionsHeightPercent != def.PositionsHeightPercent {
		t.Errorf("PositionsHeightPercent должен остаться дефолтным (%d), получено %d", def.PositionsHeightPercent, cfg.PositionsHeightPercent)
	}
	if cfg.NewsHeightRows != def.NewsHeightRows {
		t.Errorf("NewsHeightRows должен остаться дефолтным (%d), получено %d", def.NewsHeightRows, cfg.NewsHeightRows)
	}
}

func TestLoadLayoutSettings_InvalidYAMLReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "layout.yaml")
	if err := os.WriteFile(path, []byte("not: [valid: yaml"), 0644); err != nil {
		t.Fatalf("не удалось создать тестовый файл: %v", err)
	}
	if _, err := LoadLayoutSettings(path); err == nil {
		t.Error("LoadLayoutSettings() должен вернуть ошибку на невалидном YAML, не молчать")
	}
}

func TestLoadLayoutSettings_OutOfRangePercentReturnsError(t *testing.T) {
	cases := []string{
		"rightbar_width_percent: 0\n",
		"rightbar_width_percent: 100\n",
		"rightbar_width_percent: -5\n",
		"positions_height_percent: 150\n",
	}
	for _, content := range cases {
		path := filepath.Join(t.TempDir(), "layout.yaml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("не удалось создать тестовый файл: %v", err)
		}
		if _, err := LoadLayoutSettings(path); err == nil {
			t.Errorf("LoadLayoutSettings() с содержимым %q должен вернуть ошибку валидации", content)
		}
	}
}

func TestLoadLayoutSettings_NegativeNewsHeightReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "layout.yaml")
	if err := os.WriteFile(path, []byte("news_height_rows: -1\n"), 0644); err != nil {
		t.Fatalf("не удалось создать тестовый файл: %v", err)
	}
	if _, err := LoadLayoutSettings(path); err == nil {
		t.Error("LoadLayoutSettings() с отрицательным news_height_rows должен вернуть ошибку")
	}
}

func TestLayoutSettings_RightbarWidth(t *testing.T) {
	s := LayoutSettings{RightbarWidthPercent: 40}
	if got := s.rightbarWidth(160); got != 64 {
		t.Errorf("rightbarWidth(160) = %d, ожидалось 64 (40%% от 160)", got)
	}
}

func TestLayoutSettings_PositionsHeight(t *testing.T) {
	s := LayoutSettings{PositionsHeightPercent: 40}
	if got := s.positionsHeight(50); got != 20 {
		t.Errorf("positionsHeight(50) = %d, ожидалось 20 (40%% от 50)", got)
	}
}
