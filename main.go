package main

import (
	"fmt"
	tgbotapi "github.com/Syfaro/telegram-bot-api"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"reflect"
	"strings"
)

const dbDialect = "sqlite3"
const dbPath = "football.db"

//var dbAll *gorm.DB

type FootballList struct {
	//gorm.Model
	ID    int `gorm:"primary_key"`
	Plus  string
	Minus string
	Maybe string
}

func openDb() (*gorm.DB, error) {
	db, err := gorm.Open(dbDialect, dbPath)
	return db, err
}

func initialMigration() {
	db, err := openDb()
	if err != nil {
		fmt.Println(err.Error())
		panic("Fault occured while migrating")
	}
	//dbAll = db
	defer db.Close()
	db.AutoMigrate(&FootballList{})
	fmt.Println("Initial migration completed.")
}

func createNewList() error {
	db, err := openDb()
	if err != nil {
		return err
	}
	defer db.Close()

	db.Debug().Create(&FootballList{})

	return nil
}

func findLastList(db *gorm.DB, list *FootballList) error {
	db.Debug().Last(list)
	fmt.Println(&list)
	return nil
}

func deleteDuplicate(players string, name string) string {
	split := strings.Split(players, ",")

	for i, existingName := range split {
		if strings.ToLower(existingName) == strings.ToLower(name) {
			split = append(split[:i], split[i+1:]...)
			break
		}
	}
	return strings.Join(split, ",")
}

func addComma(s *string) {
	if *s != "" {
		*s += ","
	}
}

func addNewParticipant(name string, action string) error {
	db, err := openDb()
	var lastList FootballList
	if findError := findLastList(db, &lastList); findError != nil {
		return findError
	}
	if err != nil {
		return err
	}
	defer db.Close()

	lastList.Plus = deleteDuplicate(lastList.Plus, name)
	lastList.Minus = deleteDuplicate(lastList.Minus, name)
	lastList.Maybe = deleteDuplicate(lastList.Maybe, name)

	switch action {
	case "+":
		addComma(&lastList.Plus)
		lastList.Plus += name
	case "-":
		addComma(&lastList.Minus)
		lastList.Minus += name
	case "?":
		addComma(&lastList.Maybe)
		lastList.Maybe += name
	default:
		return fmt.Errorf("Unknown command.")
	}

	db.Debug().Save(&lastList)

	return nil
}

func showLastList() (*FootballList, error) {
	db, err := openDb()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	var list FootballList
	if findErr := findLastList(db, &list); findErr != nil {
		return nil, findErr
	}
	return &list, nil
}

const botToken = "922019143:AAHgtoELxHIrYNZAv5HQOuz1tTjGQ-KI2jk"

func telegramBot() {

	//Создаем бота
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected to telegram.")
	//Устанавливаем время обновления
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	//Получаем обновления от бота
	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		//Проверяем что от пользователья пришло именно текстовое сообщение
		if reflect.TypeOf(update.Message.Text).Kind() == reflect.String && update.Message.Text != "" {

			switch update.Message.Text {
			case "/start":

				//Отправлем сообщение
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Hi, i'm a football counter bot, i can create a list of players for the next game.")
				_, _ = bot.Send(msg)

			case "/new_list":

				var message string
				userName := fmt.Sprint(update.Message.From.FirstName, update.Message.From.LastName)
				if strings.Contains(userName, "IvanKharkevich") {
					if err := createNewList(); err != nil {
						message = "Что-то пошло не так и у меня не получилось создать новый список."
					} else {
						message = fmt.Sprintf("%s создал новый список", userName)
						fmt.Println(message)
					}
				} else {
					message = "У вас нет прав на создание нового списка."
				}
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
				_, _ = bot.Send(msg)
			case "/list":
				list, err := showLastList()
				var message string
				if err != nil {
					message = "Что-то пошло не так =("
				} else {
					message = fmt.Sprintf(
						"Расклад на ближайшую игру:\n- точно идут: %s\n- возможно пойдут: %s\n- не идут: %s", list.Plus, list.Maybe, list.Minus)
				}
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
				_, _ = bot.Send(msg)
			case "/info":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Справочная информация: данный бот предназначен для упрощения записи на игру. \nЧтобы отметить, что Вы хотите пойти на игру, отправьте в чат \"+\". \nЕсли Вы еще не уверены, сможете или нет, то отправьте \"?\". \nЕсли точно не получится придти на ближайшую игру, то отправьте \"-\".")
				_, _ = bot.Send(msg)
			default:
				var message string

				if err := addNewParticipant(fmt.Sprintf("%s %s", string(update.Message.From.LastName), string(update.Message.From.FirstName)), string(update.Message.Text)); err != nil {
					message = "Для записи используйте \"+\",\"-\" или \"?\""
				} else {
					message = "Добавил"
				}
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
				_, _ = bot.Send(msg)
			}
		} else {

			//Отправлем сообщение
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Используйте кириллицу.")
			_, _ = bot.Send(msg)
		}
	}
}

func main() {
	initialMigration()
	//time.Sleep(1 * time.Minute)
	//Вызываем бота
	telegramBot()
}
