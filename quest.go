package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

// ====================== СТРУКТУРЫ ======================

type Quest struct {
	ID          int       `json:"id"`
	Type        string    `json:"type"` // daily / weekly / monthly
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Deadline    time.Time `json:"deadline"`
	XP          int       `json:"xp"`
	Gold        int       `json:"gold"`
	Completed   bool      `json:"completed"`
}

type Player struct {
	Level     int       `json:"level"`
	TotalXP   int       `json:"totalXP"`
	Gold      int       `json:"gold"`
	Streak    int       `json:"streak"`
	LastDaily time.Time `json:"lastDaily"` // для расчёта streak
}

type Data struct {
	Player      Player    `json:"player"`
	Quests      []Quest   `json:"quests"`
	LastDaily   time.Time `json:"lastDailyReset"`
	LastWeekly  time.Time `json:"lastWeeklyReset"`
	LastMonthly time.Time `json:"lastMonthlyReset"`
}

const dataFile = "questhero_data.json"

// ====================== ШАБЛОНЫ КВЕСТОВ (как в MMO) ======================

var templates = map[string][]struct {
	Title string
	Desc  string
	XP    int
	Gold  int
}{
	"daily": {
		{"Сделать утреннюю зарядку", "10–15 минут любой активности", 50, 10},
		{"Выпить 2 литра воды", "Следить за гидратацией весь день", 40, 8},
		{"Прочитать 10 страниц книги", "Художественной или полезной", 35, 7},
		{"Пройти 10 000 шагов", "Прогулка или бег", 60, 12},
		{"Медитировать 10 минут", "Через приложение или просто", 30, 6},
		{"Выучить 5 новых слов", "Английский или любой язык", 45, 9},
		{"Сделать уборку в комнате", "Полная уборка рабочего места", 40, 8},
		{"Приготовить здоровый ужин", "Без фастфуда", 55, 11},
		{"Написать 3 благодарности", "В дневник или заметки", 25, 5},
		{"Изучить новый навык 20 минут", "Кодинг, гитара, рисование и т.д.", 50, 10},
	},
	"weekly": {
		{"Сходить в спортзал 3 раза", "Или любая тренировка", 200, 40},
		{"Прочитать 100 страниц книги", "Или закончить главу", 150, 30},
		{"Приготовить 5 разных блюд", "Экспериментировать на кухне", 180, 35},
		{"Сделать генеральную уборку дома", "Весь дом или квартира", 170, 35},
		{"Изучить новую тему 2 часа", "Курс, статья, видео", 160, 30},
		{"Пройти 50 км за неделю", "Бег/ходьба/велосипед", 220, 45},
	},
	"monthly": {
		{"Прочитать целую книгу", "Любую книгу от начала до конца", 500, 100},
		{"Сэкономить 5000 руб", "На любой цели", 400, 80},
		{"Выучить 50 новых слов", "Или грамматику языка", 450, 90},
		{"Сбросить/набрать 2 кг", "С помощью спорта и питания", 600, 120},
		{"Освоить новый навык", "Например, базовый Python / гитара", 550, 110},
	},
}

// ====================== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ======================

func loadData() Data {
	data := Data{
		Player: Player{Level: 1, Gold: 100},
	}
	file, err := os.ReadFile(dataFile)
	if err == nil {
		json.Unmarshal(file, &data)
	}
	return data
}

func saveData(d Data) {
	file, _ := json.MarshalIndent(d, "", "  ")
	os.WriteFile(dataFile, file, 0644)
}

// Проверка, тот ли же день/неделя/месяц
func sameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func sameWeek(t1, t2 time.Time) bool {
	y1, w1 := t1.ISOWeek()
	y2, w2 := t2.ISOWeek()
	return y1 == y2 && w1 == w2
}

func sameMonth(t1, t2 time.Time) bool {
	y1, m1, _ := t1.Date()
	y2, m2, _ := t2.Date()
	return y1 == y2 && m1 == m2
}

// Дедлайн до конца дня/недели/месяца
func endOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, t.Location())
}

func endOfWeek(t time.Time) time.Time {
	// До следующего воскресенья (или сегодня, если уже воскресенье)
	weekday := int(t.Weekday()) // 0 = Sunday
	daysToAdd := (7 - weekday) % 7
	if daysToAdd == 0 {
		daysToAdd = 7
	}
	return t.AddDate(0, 0, daysToAdd).Truncate(24 * time.Hour)
}

func endOfMonth(t time.Time) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m+1, 1, 0, 0, 0, 0, t.Location())
}

// ====================== ГЕНЕРАЦИЯ КВЕСТОВ ======================

func generateQuests(t string, count int, now time.Time) []Quest {
	pool, exists := templates[t]
	if !exists || len(pool) == 0 {
		return nil
	}

	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	quests := make([]Quest, 0, count)
	id := 1 // временный, потом перезапишем глобальным
	for i := 0; i < count && i < len(pool); i++ {
		tpl := pool[i]
		var deadline time.Time
		switch t {
		case "daily":
			deadline = endOfDay(now)
		case "weekly":
			deadline = endOfWeek(now)
		case "monthly":
			deadline = endOfMonth(now)
		}

		quests = append(quests, Quest{
			ID:          id, // будет перезаписан
			Type:        t,
			Title:       tpl.Title,
			Description: tpl.Desc,
			Deadline:    deadline,
			XP:          tpl.XP,
			Gold:        tpl.Gold,
			Completed:   false,
		})
		id++
	}
	return quests
}

func refreshQuests(data *Data, now time.Time) {
	// Daily
	if data.LastDaily.IsZero() || !sameDay(data.LastDaily, now) {
		newDaily := generateQuests("daily", 4, now) // 4 ежедневных квеста
		data.Quests = append(data.Quests, newDaily...)
		data.LastDaily = now
	}

	// Weekly
	if data.LastWeekly.IsZero() || !sameWeek(data.LastWeekly, now) {
		newWeekly := generateQuests("weekly", 2, now)
		data.Quests = append(data.Quests, newWeekly...)
		data.LastWeekly = now
	}

	// Monthly
	if data.LastMonthly.IsZero() || !sameMonth(data.LastMonthly, now) {
		newMonthly := generateQuests("monthly", 1, now)
		data.Quests = append(data.Quests, newMonthly...)
		data.LastMonthly = now
	}
}

// ====================== ОСНОВНЫЕ ФУНКЦИИ ======================

func showQuests(data Data) {
	fmt.Println("\n=== АКТИВНЫЕ КВЕСТЫ ===")
	active := false
	for i, q := range data.Quests {
		if q.Completed {
			continue
		}
		if time.Now().After(q.Deadline) {
			fmt.Printf("%d. [ПРОСРОЧЕН] %s (%s) — %s\n", i+1, q.Title, q.Type, q.Deadline.Format("02.01 15:04"))
			continue
		}
		active = true
		status := "⏳"
		fmt.Printf("%d. %s %s (%s) — до %s\n   %s\n   Награда: %d XP + %d Gold\n",
			i+1, status, q.Title, q.Type, q.Deadline.Format("02.01 15:04"), q.Description, q.XP, q.Gold)
	}
	if !active {
		fmt.Println("Все квесты выполнены! 🔥")
	}
}

func completeQuest(data *Data, idx int) {
	if idx < 0 || idx >= len(data.Quests) || data.Quests[idx].Completed {
		fmt.Println("❌ Неверный ID или квест уже выполнен")
		return
	}
	q := &data.Quests[idx]
	if time.Now().After(q.Deadline) {
		fmt.Println("❌ Квест просрочен!")
		return
	}

	q.Completed = true
	data.Player.TotalXP += q.XP
	data.Player.Gold += q.Gold

	// Обновляем streak (если daily)
	if q.Type == "daily" && sameDay(data.Player.LastDaily, time.Now()) == false {
		data.Player.Streak++
		data.Player.LastDaily = time.Now()
	} else if q.Type == "daily" {
		data.Player.LastDaily = time.Now()
	}

	// Уровень
	newLevel := 1 + data.Player.TotalXP/800 // каждые 800 XP = +1 уровень
	if newLevel > data.Player.Level {
		fmt.Printf("🎉 LEVEL UP! Новый уровень: %d\n", newLevel)
		data.Player.Level = newLevel
	}

	fmt.Printf("✅ Квест выполнен! +%d XP, +%d Gold\n", q.XP, q.Gold)
}

func showStats(data Data) {
	fmt.Println("\n=== СТАТИСТИКА ГЕРОЯ ===")
	fmt.Printf("Уровень: %d\n", data.Player.Level)
	fmt.Printf("Всего XP: %d\n", data.Player.TotalXP)
	fmt.Printf("Золото: %d\n", data.Player.Gold)
	fmt.Printf("Текущий стрик дней: %d 🔥\n", data.Player.Streak)
	fmt.Printf("Выполнено квестов всего: %d\n", countCompleted(data))
}

func countCompleted(data Data) int {
	c := 0
	for _, q := range data.Quests {
		if q.Completed {
			c++
		}
	}
	return c
}

func addCustomQuest(data *Data) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Тип квеста (daily/weekly/monthly): ")
	typ, _ := reader.ReadString('\n')
	typ = strings.TrimSpace(typ)

	fmt.Print("Название: ")
	title, _ := reader.ReadString('\n')
	title = strings.TrimSpace(title)

	fmt.Print("Описание: ")
	desc, _ := reader.ReadString('\n')
	desc = strings.TrimSpace(desc)

	fmt.Print("XP награда: ")
	var xp int
	fmt.Scanln(&xp)

	fmt.Print("Gold награда: ")
	var gold int
	fmt.Scanln(&gold)

	now := time.Now()
	var deadline time.Time
	switch typ {
	case "daily":
		deadline = endOfDay(now)
	case "weekly":
		deadline = endOfWeek(now)
	case "monthly":
		deadline = endOfMonth(now)
	default:
		fmt.Println("❌ Неизвестный тип!")
		return
	}

	data.Quests = append(data.Quests, Quest{
		ID:          len(data.Quests) + 1,
		Type:        typ,
		Title:       title,
		Description: desc,
		Deadline:    deadline,
		XP:          xp,
		Gold:        gold,
		Completed:   false,
	})
	fmt.Println("✅ Кастомный квест добавлен!")
}

// ====================== MAIN ======================

func main() {
	rand.Seed(time.Now().UnixNano())
	data := loadData()
	now := time.Now()

	// Автогенерация квестов
	refreshQuests(&data, now)

	// Пересчитываем ID (на всякий случай)
	for i := range data.Quests {
		data.Quests[i].ID = i + 1
	}

	fmt.Println("🌟 QuestHero запущен! Добро пожаловать, герой!")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("\n=== МЕНЮ ===")
		fmt.Println("1. Показать квесты")
		fmt.Println("2. Завершить квест (номер)")
		fmt.Println("3. Статистика")
		fmt.Println("4. Добавить свой квест")
		fmt.Println("5. Сгенерировать новые квесты (принудительно)")
		fmt.Println("0. Выход")
		fmt.Print("Выберите действие: ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			showQuests(data)
		case "2":
			fmt.Print("Номер квеста: ")
			var id int
			fmt.Scanln(&id)
			completeQuest(&data, id-1)
			saveData(data)
		case "3":
			showStats(data)
		case "4":
			addCustomQuest(&data)
			saveData(data)
		case "5":
			fmt.Println("Генерируем свежие квесты...")
			refreshQuests(&data, now)
			saveData(data)
			fmt.Println("✅ Готово!")
		case "0":
			saveData(data)
			fmt.Println("До встречи, герой! 💪")
			return
		default:
			fmt.Println("Неверный выбор")
		}
	}
}
